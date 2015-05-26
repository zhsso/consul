package consul

import (
	"fmt"
	"time"

	consulApi "github.com/hashicorp/consul/api"
)

const (
	TTL              = 16
	WAIT_TIME        = 60 * time.Second
	ERROR_SLEEP_TIME = 4 * time.Second
)

type ServiceAgent struct {
	consulClient    *consulApi.Client
	consulLastIndex uint64
	serviceName     string
	services        []*consulApi.ServiceEntry
	callBack        func([]*consulApi.ServiceEntry)
	running         bool
}

func NewServiceAgent(serviceName, address string) (service *ServiceAgent, err error) {
	cnf := consulApi.DefaultConfig()
	cnf.WaitTime = WAIT_TIME
	if address != "" {
		cnf.Address = address
	}
	consulClient, err := consulApi.NewClient(cnf)
	if err != nil {
		return nil, err
	}

	service = &ServiceAgent{
		serviceName:  serviceName,
		running:      true,
		callBack:     func([]*consulApi.ServiceEntry) {},
		consulClient: consulClient,
	}

	go service.agentDaemon()
	return
}

func (s *ServiceAgent) Stop() {
	s.running = false
}

func (s *ServiceAgent) serviceExists(serviceId string) (ok bool, err error) {
	agent := s.consulClient.Agent()
	services, err := agent.Services()
	if err != nil {
		return false, err
	}
	_, ok = services[serviceId]
	return
}

func (s *ServiceAgent) UnregisterService(serviceId string) error {
	agent := s.consulClient.Agent()
	return agent.ServiceDeregister(serviceId)
}

//注册服务
func (s *ServiceAgent) RegisterService(serviceId, address string, port int, tags ...string) (err error) {
	if ok, _ := s.serviceExists(serviceId); ok {
		if err = s.UnregisterService(serviceId); err != nil {
			return
		}
	}

	agent := s.consulClient.Agent()
	reg := &consulApi.AgentServiceRegistration{
		ID:      serviceId,
		Name:    s.serviceName,
		Tags:    tags,
		Port:    port,
		Address: address,
		Check: &consulApi.AgentServiceCheck{
			TTL: fmt.Sprintf("%ds", TTL),
		},
	}
	err = agent.ServiceRegister(reg)
	if err != nil {
		return err
	}
	tk := time.NewTicker(time.Second * (TTL / 2))
	checkId := fmt.Sprintf("service:%s", serviceId)
	go func() {
		var now time.Time
		for range tk.C {
			now = time.Now()
			agent.PassTTL(checkId, now.String())
		}
		tk.Stop()
	}()

	return
}

func (s *ServiceAgent) SetCallBack(a func([]*consulApi.ServiceEntry)) {
	s.callBack = a
}

func (s *ServiceAgent) GetServices() []*consulApi.ServiceEntry {
	return s.services
}

func (s *ServiceAgent) agentDaemon() {
	h := s.consulClient.Health()
	for s.running {
		idx := s.consulLastIndex
		q := &consulApi.QueryOptions{
			WaitIndex: idx,
		}
		//return service that status is passing.
		services, meta, err := h.Service(s.serviceName, "", true, q)
		if err != nil || meta.LastIndex == 0 {
			time.Sleep(ERROR_SLEEP_TIME)
			continue
		}

		if idx != meta.LastIndex {
			s.consulLastIndex = meta.LastIndex
			s.services = services
			s.callBack(services)
		}
	}
}
