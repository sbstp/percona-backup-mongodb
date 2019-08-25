package pbm

import (
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type Node struct {
	name string
	cn   *mongo.Client
	opts string
}

func NewNode(name string, conn *mongo.Client, curi string) *Node {
	return &Node{
		name: name,
		cn:   conn,
		opts: curi,
	}
}

func (n *Node) GetIsMaster() (*IsMaster, error) {
	im := &IsMaster{}
	err := n.cn.Database(DB).RunCommand(nil, bson.D{{"isMaster", 1}}).Decode(im)
	if err != nil {
		return nil, errors.Wrap(err, "run mongo command isMaster")
	}
	return im, nil
}

func (n *Node) Name() (string, error) {
	im, err := n.GetIsMaster()
	if err != nil {
		return "", err
	}
	return im.Me, nil
}

func (n *Node) GetReplsetStatus() (*ReplsetStatus, error) {
	status := &ReplsetStatus{}
	err := n.cn.Database(DB).RunCommand(nil, bson.D{{"replSetGetStatus", 1}}).Decode(status)
	if err != nil {
		return nil, errors.Wrap(err, "run mongo command replSetGetStatus")
	}
	return status, err
}

func (n *Node) Status() (*NodeStatus, error) {
	s, err := n.GetReplsetStatus()
	if err != nil {
		return nil, err
	}

	name, err := n.Name()
	if err != nil {
		return nil, errors.Wrap(err, "get node name")
	}

	for _, m := range s.Members {
		if m.Name == name {
			return &m, nil
		}
	}

	return nil, errors.New("not found")
}

func (n *Node) ConnURI() string {
	return n.opts
}