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

package driver

//"time"
//"fmt"

//"github.com/go-ble/ble"
//"github.com/go-ble/ble/linux"
//"golang.org/x/net/context"
//"k8s.io/klog/v2"
//"github.com/kubeedge/mappers-go/mappers/common"
//this is a test
//this is a ets
type BleConfig struct {
	Addr string
}

/*
func (bc *BleClient) Set(c ble.UUID, b []byte) error {
	if p, err := bc.Client.DiscoverProfile(true); err == nil {
		//fmt.Println("client.go 63 falg")
		if u := p.Find(ble.NewCharacteristic(c)); u != nil {
			c := u.(*ble.Characteristic)
			//fmt.Println("client.go 66 flag")
			if err := bc.Client.WriteCharacteristic(c, b, false); err != nil {
				//fmt.Println("client.go 68 flag")
				klog.Errorf("Write characteristic error %v", err)
				return err
			}
		}
	}
	return nil
}

func (bc *BleClient) Read(c *ble.Characteristic) ([]byte, error) {
	return bc.Client.ReadCharacteristic(c)
}

/*
// GetStatus get device status.
// Now we could only get the connection status.
func (bc *BleClient) GetStatus() string {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	rssi := bc.Client.ReadRSSI()
	if rssi < 0 {
		return common.DEVSTOK
	}
	return common.DEVSTDISCONN

	return common.DEVSTOK
}
*/
