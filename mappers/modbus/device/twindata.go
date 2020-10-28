/*
Copyright 2020 The KubeEdge Authors.

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

package dev

import (
	"strconv"
	"strings"

	mappercommon "github.com/kubeedge/kubeedge/mappers/common"
	"github.com/kubeedge/kubeedge/mappers/modbus/driver"
	"github.com/kubeedge/kubeedge/mappers/modbus/globals"
	"k8s.io/klog"
)

// TwinData is the timer structure for getting twin/data.
type TwinData struct {
	Client       driver.ModbusClient
	Name         string
	Type         string
	RegisterType string
	Address      uint16
	Quantity     uint16
	Results      []byte
	Topic        string
}

// Run timer function.
func (td *TwinData) Run() {
	var err error
	td.Results, err = td.Client.Get(td.RegisterType, td.Address, td.Quantity)
	if err != nil {
		klog.Error("Get register failed: ", err)
		return
	}
	// construct payload
	var payload []byte
	if strings.Contains(td.Topic, "update") {
		if payload, err = mappercommon.CreateMessageTwinUpdate(td.Name, td.Type, strconv.Itoa(int(td.Results[0]))); err != nil {
			klog.Error("Create message twin update failed")
			return
		}
		klog.Info("Update value:", string(int(td.Results[0])))
	} else {
		if payload, err = mappercommon.CreateMessageData(td.Name, td.Type, string(td.Results)); err != nil {
			klog.Error("Create message data failed")
			return
		}
	}
	if err = globals.MqttClient.Publish(td.Topic, payload); err != nil {
		klog.Error(err)
	}
}
