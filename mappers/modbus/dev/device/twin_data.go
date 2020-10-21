package device

import (
	"encoding/json"
	mappercommon "github.com/kubeedge/kubeedge/mappers/common"
	"github.com/kubeedge/kubeedge/mappers/modbus/dev"
	"github.com/kubeedge/kubeedge/mappers/modbus/globals"
	"k8s.io/klog"
	"strconv"
	"strings"
)

type TwinData struct {
	Client       globals.ModbusClient
	Name         string
	Type         string
	RegisterType string
	Address      uint16
	Quantity     uint16
	Results      []byte
	Topic        string
}

func (td *TwinData) Run() {
	var err error
	td.Results, err = dev.Get(td.Client, td.RegisterType, td.Address, td.Quantity)
	if err != nil {
		klog.Error("Get register failed")
		return
	}
	// construct payload
	var payload []byte
	if strings.Contains(td.Topic, "update") {
		payload = mappercommon.CreateMessageTwinUpdate(td.Name, td.Type, strconv.Itoa(int(td.Results[0])))
		var deviceTwinUpdate mappercommon.DeviceTwinUpdate
		err = json.Unmarshal(payload, &deviceTwinUpdate)
		if err != nil {
			klog.Error("Unmarshal error")
			return
		}
	} else {
		payload = mappercommon.CreateMessageData(td.Name, td.Type, string(td.Results))
		var data mappercommon.DeviceData
		err = json.Unmarshal(payload, &data)
		klog.Error(err, data)
	}
	err = globals.MqttClient.Publish(td.Topic, payload)

	if err != nil {
		klog.Error(err)
	}
}
