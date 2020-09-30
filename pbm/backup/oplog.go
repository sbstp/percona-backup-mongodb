package backup

import (
	"context"
	"io"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/sbstp/percona-backup-mongodb/pbm"
)

// Oplog is used for reading the Mongodb oplog
type Oplog struct {
	node  *pbm.Node
	start primitive.Timestamp
	end   primitive.Timestamp
}

// NewOplog creates a new Oplog instance
func NewOplog(node *pbm.Node) *Oplog {
	return &Oplog{
		node: node,
	}
}

// SetTailingSpan sets oplog tailing window
func (ot *Oplog) SetTailingSpan(start, end primitive.Timestamp) {
	ot.start = start
	ot.end = end
}

// WriteTo writes an oplog slice between start and end timestamps into the given io.Writer
//
// To be sure we have read ALL records up to the specified cluster time.
// Specifically, to be sure that no operations from the past gonna came after we finished the slicing,
// we have to tail until some record with ts > endTS. And it might be a noop.
func (ot *Oplog) WriteTo(w io.Writer) (int64, error) {
	if ot.start.T == 0 || ot.end.T == 0 {
		return 0, errors.Errorf("oplog TailingSpan should be set, have start: %v, end: %v", ot.start, ot.end)
	}

	ctx := context.Background()

	clName, err := ot.collectionName()
	if err != nil {
		return 0, errors.Wrap(err, "determine oplog collection name")
	}
	cl := ot.node.Session().Database("local").Collection(clName)

	cur, err := cl.Find(ctx,
		bson.M{
			"ts": bson.M{"$gte": ot.start},
		},
		options.Find().SetCursorType(options.Tailable),
	)
	if err != nil {
		return 0, errors.Wrap(err, "get the oplog cursor")
	}
	defer cur.Close(ctx)

	opts := primitive.Timestamp{}
	var ok bool
	var written int64
	for cur.Next(ctx) {
		opts.T, opts.I, ok = cur.Current.Lookup("ts").TimestampOK()
		if !ok {
			return written, errors.Errorf("get the timestamp of record %v", cur.Current)
		}
		if primitive.CompareTimestamp(ot.end, opts) == -1 {
			return written, nil
		}

		// skip noop operations
		if cur.Current.Lookup("op").String() == string(pbm.OperationNoop) {
			continue
		}

		n, err := w.Write([]byte(cur.Current))
		if err != nil {
			return written, errors.Wrap(err, "write to pipe")
		}
		written += int64(n)
	}

	return written, cur.Err()
}

var errMongoTimestampNil = errors.New("timestamp is nil")

// LastWrite returns a timestamp of the last write operation readable by majority reads
func (ot *Oplog) LastWrite() (primitive.Timestamp, error) {
	isMaster, err := ot.node.GetIsMaster()
	if err != nil {
		return primitive.Timestamp{}, errors.Wrap(err, "get isMaster data")
	}
	if isMaster.LastWrite.MajorityOpTime.TS.T == 0 {
		return primitive.Timestamp{}, errMongoTimestampNil
	}
	return isMaster.LastWrite.MajorityOpTime.TS, nil
}

func (ot *Oplog) collectionName() (string, error) {
	isMaster, err := ot.node.GetIsMaster()
	if err != nil {
		return "", errors.Wrap(err, "get isMaster document")
	}

	if len(isMaster.Hosts) > 0 {
		return "oplog.rs", nil
	}
	if !isMaster.IsMaster {
		return "", errors.New("not connected to master")
	}
	return "oplog.$main", nil
}
