/*
Copyright 2021 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package device

import (
	"encoding/json"
	"fmt"
	"sort"

	"flag"
	"sync"
	"time"

	"github.com/go-ble/ble"
	"github.com/go-ble/ble/examples/lib/dev"

	"k8s.io/klog/v2"

	"github.com/kubeedge/mappers-go/mappers/ble/configmap"
	"github.com/kubeedge/mappers-go/mappers/ble/driver"
	"github.com/kubeedge/mappers-go/mappers/ble/globals"
	"github.com/kubeedge/mappers-go/mappers/common"
)

var devices map[string]*globals.BleDev
var models map[string]common.DeviceModel
var protocols map[string]common.Protocol
var wg sync.WaitGroup
var (
	device = flag.String("device", "default", "implementation of ble")
)

/*
// setVisitor check if visitory property is readonly, if not then set it.twin的desire设置
func SetVisitor(bleClient ) {

	err := bleClient.Set(ble.MustParse("6E400009B5A3F393E0A9E50E24DCCA9E"), []byte{1})
	if err != nil {
		klog.Errorf("Set visitor error: %v %c", err, "6E400009B5A3F393E0A9E50E24DCCA9E")
		return
	}
}
*/

//obtain information about all devices
func getinformation() Messagelist {
	message := []TwinData{}
	addr := []driver.BleConfig{}
	collecttime := []time.Duration{}

	//the device is special so we should arrange the device connection sequence
	sequence := []string{}
	for key := range devices {
		sequence = append(sequence, key)
	}
	sort.Strings(sequence)
	for k := 0; k < len(sequence); k++ {
		dev := devices[sequence[k]]
		//init
		property := []PropertyData{}
		//MAC address
		var protocolConfig configmap.BleProtocolConfig
		if err := json.Unmarshal([]byte(dev.Instance.PProtocol.ProtocolConfigs), &protocolConfig); err != nil {
			klog.Errorf("Unmarshal ProtocolConfig error: %v", err)
		}
		config := driver.BleConfig{
			Addr: protocolConfig.MacAddress,
		}
		addr = append(addr, config)

		//all twin property
		for i := 0; i < len(dev.Instance.Twins); i++ {
			var visitorConfig configmap.BleVisitorConfig
			if err := json.Unmarshal(dev.Instance.Twins[i].PVisitor.VisitorConfig, &visitorConfig); err != nil {
				klog.Errorf("Unmarshal VisitorConfig error: %v", err)
				continue
			}
			propertyData := PropertyData{
				Name:             dev.Instance.Twins[i].PropertyName,
				Type:             dev.Instance.Twins[i].Desired.Metadatas.Type,
				BleVisitorConfig: visitorConfig,
			}
			property = append(property, propertyData)
		}

		//collectCycle
		collect := time.Duration(dev.Instance.Twins[0].PVisitor.CollectCycle)
		if collect == 0 {
			collect = 5 * time.Second
		}
		collecttime = append(collecttime, collect)

		//twindata
		twinData := TwinData{
			ID:       dev.Instance.ID,
			Property: property,
			Topic:    fmt.Sprintf(common.TopicTwinUpdate, dev.Instance.ID)}

		message = append(message, twinData)
	}
	messagelist := Messagelist{
		Address:      addr,
		CollectCycle: collecttime,
		Message:      message}
	return messagelist
}

// initTwin
func initTwin(message Messagelist) {
	message.Run()
}

// initGetStatus start timer to get device status and send to eventbus.
func initGetStatus(dev *globals.BleDev) {
	getStatus := GetStatus{topic: fmt.Sprintf(common.TopicStateUpdate, dev.Instance.ID),
		ID: dev.Instance.ID}
	timer := common.Timer{Function: getStatus.Run, Duration: 10 * time.Second, Times: 0}
	wg.Add(1)
	go func() {
		defer wg.Done()
		timer.Start()
	}()
}

// DevInit initialize the device datas.
func DevInit(configmapPath string) error {
	devices = make(map[string]*globals.BleDev)
	models = make(map[string]common.DeviceModel)
	protocols = make(map[string]common.Protocol)
	return configmap.Parse(configmapPath, devices, models, protocols)
}

// DevStart start all devices.
func DevStart() {
	flag.Parse()
	setting := ble.OptDeviceID(0)
	host, err := dev.NewDevice(*device, setting)
	if err != nil {
		klog.Errorf("New device error: %v", err)
		return
	}
	ble.SetDefaultDevice(host)

	messagelist := getinformation()
	initTwin(messagelist)
	for _, dev := range devices {
		initGetStatus(dev)
	}

	wg.Wait()
}
