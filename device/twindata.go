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
	"fmt"

	"math"
	"strings"
	"time"

	"github.com/go-ble/ble"
	"golang.org/x/net/context"
	"k8s.io/klog/v2"

	"github.com/kubeedge/mappers-go/mappers/ble/configmap"
	"github.com/kubeedge/mappers-go/mappers/ble/driver"
	"github.com/kubeedge/mappers-go/mappers/ble/globals"
	"github.com/kubeedge/mappers-go/mappers/common"
)

// TwinData is the timer structure for getting twin/data.
type TwinData struct {
	ID         string
	Client     ble.Client
	Topic      string
	Property   []PropertyData
	TwinName   []string
	TwinResult []string
	TwinType   []string
}

type PropertyData struct {
	Name             string
	Type             string
	BleVisitorConfig configmap.BleVisitorConfig
	Result           string
}

type Messagelist struct {
	Address      []driver.BleConfig
	CollectCycle []time.Duration
	Message      []TwinData
}

var State map[string]bool

// Run timer function.
func (mes *Messagelist) Run() {
	fmt.Printf("\n---start connect with device ---\n")
	//all device
	State = make(map[string]bool)
	for i := len(mes.Message) - 1; i >= 0; i-- {

		//connect with ble device

		filter := func(a ble.Advertisement) bool {
			return strings.EqualFold(strings.ToUpper(a.Addr().String()), strings.ToUpper(mes.Address[i].Addr))
		}

		fmt.Printf("\nScanning for %s. with [ %s ],[ %s ]\n", 6*time.Second, mes.Message[i].ID, mes.Address[i].Addr)
		ctx := ble.WithSigHandler(context.WithTimeout(context.Background(), 6*time.Second))
		cln, err := ble.Connect(ctx, filter)

		if err != nil {
			klog.Errorf("connnect error: %v", err)
			fmt.Printf("\n---sleep for a while---\n")

			time.Sleep(5 * time.Second)
			State[mes.Message[i].ID] = false
			continue
		}
		State[mes.Message[i].ID] = true
		fmt.Printf("device %s connect successful\n", mes.Address[i].Addr)

		fmt.Printf("\n---sleep for a while---\n")
		time.Sleep(5 * time.Second)

		fmt.Printf("\n --set register-- \n")
		//init set register
		if p, err := cln.DiscoverProfile(true); err == nil {
			if u := p.Find(ble.NewCharacteristic(ble.MustParse("6E400009B5A3F393E0A9E50E24DCCA9E"))); u != nil {
				c := u.(*ble.Characteristic)
				//fmt.Println("client.go 66 flag")
				if err := cln.WriteCharacteristic(c, []byte{1}, false); err != nil {
					//fmt.Println("client.go 68 flag")
					klog.Errorf("Write characteristic error %v", err)
				}
			}
		}
		mes.Message[i].Client = cln
		//
		fmt.Printf("\n---sleep for a while---\n")
		time.Sleep(8 * time.Second)
	}

	fmt.Printf("\n --- open notify ---\n")
	for i := 0; i < len(mes.Message); i++ {
		if mes.Message[i].Client == nil {
			State[mes.Message[i].Topic] = false
			continue
		}
		for j := 0; j < len(mes.Message[i].Property); j++ {
			if p, err := mes.Message[i].Client.DiscoverProfile(true); err == nil {
				uuid := ble.MustParse(mes.Message[i].Property[j].BleVisitorConfig.CharacteristicUUID)
				findcha := p.Find(ble.NewCharacteristic(uuid))
				if findcha == nil {
					klog.Errorf("can't find uuid %s", uuid.String())
					continue
				}
				c := findcha.(*ble.Characteristic)
				if (c.Property&ble.CharNotify) != 0 && c.CCCD != nil {
					fmt.Printf("-- Subscribe to notification for [ %s ] --\n", mes.Message[i].ID)
					if err := mes.Message[i].Client.Subscribe(c, false, mes.Message[i].Property[j].notificationHandler()); err != nil {
						klog.Errorf("subscribe failed: %s", err)
					}
				}
			}
		}
		fmt.Printf("\n---sleep for a while---\n")
		time.Sleep(2 * time.Second)
	}

	fmt.Printf("\n---start publish data from device ---\n")
	for i := 0; i < len(mes.Message); i++ {
		if mes.Message[i].Client == nil {
			continue
		}
		timer := common.Timer{Function: mes.Message[i].Run, Duration: mes.CollectCycle[i], Times: 0}
		wg.Add(1)
		go func() {
			defer wg.Done()
			timer.Start()
		}()

	}

}

func (td *TwinData) Run() {

	fmt.Printf("\n---merge data ---\n")
	td.MergeData()
	fmt.Printf("\n--- publish to mqtt ---\n")
	td.handlerPublish()
	td.TwinName = td.TwinName[0:0]
	td.TwinResult = td.TwinResult[0:0]
	td.TwinType = td.TwinType[0:0]

}

func (pd *PropertyData) notificationHandler() func(req []byte) {
	return func(req []byte) {
		pd.Result = pd.ConvertReadData(req)
		//fmt.Println("Name :", pd.Name, "Result: ", pd.Result)
	}
}

//one property convert
func (pd *PropertyData) ConvertReadData(req []byte) string {
	//if the []byte is null return
	if len(req) == 0 {
		return ""
	}
	//fmt.Println("twindata.go 115 data:",data)
	initialValue := []byte{}
	count := 0
	if pd.BleVisitorConfig.DataConvert.StartIndex <= pd.BleVisitorConfig.DataConvert.EndIndex {
		for index := pd.BleVisitorConfig.DataConvert.StartIndex; index <= pd.BleVisitorConfig.DataConvert.EndIndex; index++ {
			initialValue = append(initialValue, req[index])
			count++
		}
	} else {
		for index := pd.BleVisitorConfig.DataConvert.StartIndex; index >= pd.BleVisitorConfig.DataConvert.EndIndex; index-- {
			initialValue = append(initialValue, req[index])
			count++
		}
	}
	//fmt.Println("twindata.go 129 initialValue",initialValue)
	var finalResult float64
	var mediaResult float64
	finalResult = 0.0
	mediaResult = 0.0
	for index1 := 0; index1 < count; index1++ {
		mediaResult = mediaResult + math.Pow(16, float64((index1)*2))*float64(initialValue[index1])
	}
	//fmt.Println("mediaResult",mediaResult)
	//fmt.Println(td.BleVisitorConfig.DataConvert.ShiftRight)
	if pd.BleVisitorConfig.DataConvert.ShiftLeft != 0 {
		finalResult = mediaResult * math.Pow(10, float64(pd.BleVisitorConfig.DataConvert.ShiftLeft))
	} else if pd.BleVisitorConfig.DataConvert.ShiftRight != 0 {
		finalResult = mediaResult / math.Pow(10, float64(pd.BleVisitorConfig.DataConvert.ShiftRight))
	} else {
		finalResult = mediaResult
	}
	result := fmt.Sprintf("%f", finalResult)
	return result

}

func (td *TwinData) handlerPublish() (err error) {

	var payload []byte
	if strings.Contains(td.Topic, "$hw") {
		if payload, err = common.CreateMessageTwinUpdate(td.TwinName, td.TwinType, td.TwinResult); err != nil {
			klog.Errorf("Create message twin update failed: %v", err)
			return
		}
	}
	if err = globals.MqttClient.Publish(td.Topic, payload); err != nil {
		klog.Errorf("Publish topic %v failed, err: %v", td.Topic, err)
	}

	klog.V(2).Infof("Update value:, topic: %s", td.Topic)
	return
}

// ConvertReadData is the function responsible to convert the data read from the device into meaningful data.
// If currently logic of converting data is not suitbale for your device, you can change ConvertReadData function manually.
func (td *TwinData) MergeData() {
	for i := 0; i < len(td.Property); i++ {
		fmt.Printf("\n ---merge [ %s ]--- \n", td.Property[i].Name)
		td.TwinName = append(td.TwinName, td.Property[i].Name)
		td.TwinType = append(td.TwinType, td.Property[i].Type)
		td.TwinResult = append(td.TwinResult, td.Property[i].Result)
	}
}
