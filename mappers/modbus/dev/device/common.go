package device

import (
	"encoding/json"
	"github.com/kubeedge/kubeedge/mappers/modbus/dev"
	"regexp"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	mappercommon "github.com/kubeedge/kubeedge/mappers/common"

	"github.com/kubeedge/kubeedge/mappers/modbus/configmap"
	. "github.com/kubeedge/kubeedge/mappers/modbus/globals"
	"k8s.io/klog"
)

var (
	devices   map[string]*ModbusDev
	models    map[string]mappercommon.DeviceModel
	protocols map[string]mappercommon.Protocol
	wg        sync.WaitGroup
)

func DevInit(configmapPath string) error {
	devices = make(map[string]*ModbusDev)
	models = make(map[string]mappercommon.DeviceModel)
	protocols = make(map[string]mappercommon.Protocol)
	return configmap.Parse(configmapPath, devices, models, protocols)
}

func getDeviceID(topic string) (id string) {
	re := regexp.MustCompile(`hw/events/device/(.+)/twin/update/delta`)
	return re.FindStringSubmatch(topic)[1]
}

func onMessage(client mqtt.Client, message mqtt.Message) {
	klog.Error("Receive message", message.Topic())
	// TopicTwinUpdateDelta
	id := getDeviceID(message.Topic())
	if id == "" {
		klog.Error("Wrong topic")
		return
	}
	klog.Infof("Device id: %v", id)

	// Get device ID and get device instance
	var device *ModbusDev
	var ok bool
	if device, ok = devices[id]; !ok {
		klog.Error("Device not exist")
		return
	}

	// Get twin map key as the propertyName
	var delta mappercommon.DeviceTwinDelta
	err := json.Unmarshal(message.Payload(), &delta)
	if err != nil {
		klog.Errorf("Unmarshal message failed: %v", err)
		return
	}
	for key, value := range delta.Delta {
		i := 0
		for i = 0; i < len(device.Instance.Twins); i++ {
			if key == device.Instance.Twins[i].PropertyName {
				break
			}
		}
		if i == len(device.Instance.Twins) {
			klog.Error("Twin not found")
			continue
		}
		// Type transfer
		device.Instance.Twins[i].Desired.Value = value
		var r configmap.ModbusVisitorConfig
		if err := json.Unmarshal([]byte(device.Instance.Twins[i].PVisitor.VisitorConfig), &r); err != nil {
			klog.Error("Unmarshal visitor config failed")
		}
		klog.Infof("Desired value: %v", value, device.Instance.Twins[i].PVisitor.PProperty.DataType)
		v, err := mappercommon.Convert(device.Instance.Twins[i].PVisitor.PProperty.DataType, value)
		vint, ok := v.(int64)
		if err == nil && ok {
			dev.Set(device.ModbusClient, r.Register, r.Offset, uint16(vint))
		} else {
			klog.Error("Convert failed")
		}
	}
}
