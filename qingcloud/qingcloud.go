package qingcloud

// Please see qingcloud document: https://docs.qingcloud.com/index.html
// and must pay attention to your account resource quota limit.

import (
"fmt"
"io"

"gopkg.in/gcfg.v1"

"github.com/golang/glog"
qcconfig "github.com/yunify/qingcloud-sdk-go/config"
qcservice "github.com/yunify/qingcloud-sdk-go/service"
"k8s.io/kubernetes/pkg/cloudprovider"
"io/ioutil"
	"k8s.io/kubernetes/pkg/controller"
)

const (
	ProviderName = "qingcloud"
)

type Config struct {
	Global struct {
		QYConfigPath    string `gcfg:"qyConfigPath"`
		Zone			string `gcfg:"zone"`
	}
}

// A single Kubernetes cluster can run in multiple zones,
// but only within the same region (and cloud provider).
type QingCloud struct {
	instanceService      *qcservice.InstanceService
	lbService            *qcservice.LoadBalancerService
	volumeService        *qcservice.VolumeService
	jobService           *qcservice.JobService
	securityGroupService *qcservice.SecurityGroupService
	zone                 string
	selfInstance         *qcservice.Instance
}

func init() {
	cloudprovider.RegisterCloudProvider(ProviderName, func(config io.Reader) (cloudprovider.Interface, error) {
		cfg, err := readConfig(config)
		if err != nil {
			return nil, err
		}
		return newQingCloud(cfg)
	})
}

func readConfig(config io.Reader) (Config, error) {
	if config == nil {
		err := fmt.Errorf("no qingcloud provider config file given")
		return Config{}, err
	}

	var cfg Config
	err := gcfg.ReadInto(&cfg, config)
	return cfg, err
}

// newQingCloud returns a new instance of QingCloud cloud provider.
func newQingCloud(config Config) (cloudprovider.Interface, error) {
	qcConfig, err := qcconfig.NewDefault()
	if err != nil {
		return nil, err
	}
	if err = qcConfig.LoadConfigFromFilepath(config.Global.QYConfigPath); err != nil {
		return nil, err
	}

	qcService, err := qcservice.Init(qcConfig)
	if err != nil {
		return nil, err
	}
	instanceService, err := qcService.Instance(config.Global.Zone)
	if err != nil {
		return nil, err
	}
	lbService, err := qcService.LoadBalancer(config.Global.Zone)
	if err != nil {
		return nil, err
	}
	volumeService, err := qcService.Volume(config.Global.Zone)
	if err != nil {
		return nil, err
	}
	jobService, err := qcService.Job(config.Global.Zone)
	if err != nil {
		return nil, err
	}
	securityGroupService, err := qcService.SecurityGroup(config.Global.Zone)
	if err != nil {
		return nil, err
	}

	qc := QingCloud{
		instanceService:      instanceService,
		lbService:            lbService,
		volumeService:        volumeService,
		jobService:           jobService,
		securityGroupService: securityGroupService,
		zone:                 config.Global.Zone,
	}
	host, err := getHostname()
	if err != nil {
		return nil, err
	}
	ins, err := qc.GetInstanceByID(host)
	if err != nil {
		glog.Errorf("Get self instance fail, id: %s, err: %s", host, err.Error())
		return nil, err
	}
	qc.selfInstance = ins

	glog.V(1).Infof("QingCloud provider init finish, zone: %v, selfInstance: %+v", qc.zone, qc.selfInstance)

	return &qc, nil
}

// LoadBalancer returns an implementation of LoadBalancer for QingCloud.
func (qc *QingCloud) LoadBalancer() (cloudprovider.LoadBalancer, bool) {
	glog.V(4).Info("LoadBalancer() called")
	return qc, true
}

// Instances returns an implementation of Instances for QingCloud.
func (qc *QingCloud) Instances() (cloudprovider.Instances, bool) {
	glog.V(4).Info("Instances() called")
	return qc, true
}

func (qc *QingCloud) Initialize(clientBuilder controller.ControllerClientBuilder){

}

func (qc *QingCloud) Zones() (cloudprovider.Zones, bool) {
	return qc, true
}

func (qc *QingCloud) Clusters() (cloudprovider.Clusters, bool) {
	return nil, false
}

func (qc *QingCloud) Routes() (cloudprovider.Routes, bool) {
	return nil, false
}

func (qc *QingCloud) ProviderName() string {
	return ProviderName
}

// ScrubDNS filters DNS settings for pods.
func (qc *QingCloud) ScrubDNS(nameservers, searches []string) (nsOut, srchOut []string) {
	return nameservers, searches
}

func (qc *QingCloud) GetZone() (cloudprovider.Zone, error) {
	glog.V(4).Infof("GetZone() called, current zone is %v", qc.zone)

	return cloudprovider.Zone{Region: qc.zone}, nil
}

func getHostname() (string, error) {
	content, err := ioutil.ReadFile("/etc/qingcloud/instance-id")
	if err != nil {
		return "", err
	}
	return string(content), nil
}
