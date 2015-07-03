package models

import (
	//"K8APITransform/ApiServer/models"
	//"github.com/coreos/go-etcd/etcd"
	"sync"
)

var mutex = &sync.Mutex{}
var IdPools IdPoolsInterface

type IdPoolsInterface interface {
	GetId(env string) (string, error)
	CreateIdPool(env string) error
}

func NewIdPools() IdPoolsInterface {
	_, err := EtcdClient.Get("/idpools", false, false)
	if err != nil {
		EtcdClient.CreateDir("/idpools", 0)
	}
	return &idpools{}
}

type idpools struct {
}

func (pools *idpools) CreateIdPool(env string) error {
	_, err := EtcdClient.Create("/idpools/"+env, "aaaaaaaaaaaaa", 0)
	return err

}
func (pools *idpools) GetId(env string) (string, error) {
	mutex.Lock()
	defer mutex.Unlock()
	response, err := EtcdClient.Get("/idpools/"+env, false, false)
	if err != nil {
		return "", err
	}
	id := pools.next(response.Node.Value)
	_, err = EtcdClient.Update("/idpools/"+env, id, 0)
	if err != nil {
		return "", err
	}
	return response.Node.Value, nil
}
func (pool *idpools) next(Id string) string {
	id := []byte(Id)
	//t := 0
	for i := 12; i >= 0; i-- {
		if id[i] == byte('z') {
			id[i] = byte('a')
		} else {
			id[i]++
			break
		}
	}
	return string(id)
}