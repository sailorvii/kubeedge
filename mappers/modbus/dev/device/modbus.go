package device

import (
	"encoding/json"
	"fmt"
	mappercommon "github.com/kubeedge/kubeedge/mappers/common"
	"github.com/kubeedge/kubeedge/mappers/modbus/configmap"
	"github.com/kubeedge/kubeedge/mappers/modbus/dev"
	"github.com/kubeedge/kubeedge/mappers/modbus/globals"
	"k8s.io/klog"
	"strconv"
	"time"
)

func Start(md *globals.ModbusDev) (err error) {
	var pcc configmap.ModbusProtocolCommonConfig
	if err = json.Unmarshal([]byte(md.Instance.PProtocol.ProtocolCommonConfig), &pcc); err != nil {
		klog.Error(err)
		return
	}
	// TODO should valid params when parse deviceProfile.json
	// tcp and rtu are mutually exclusive
	if pcc.Com.SerialPort != "" && pcc.Tcp.Ip != "" {
		klog.Error("tcp and rtu are mutually exclusive.")
		return fmt.Errorf("tcp and rtu are mutually exclusive.")
	}

	if pcc.Com.SerialPort == "" && pcc.Tcp.Ip == "" {
		klog.Error("SerialPort and Ip are both null")
		return fmt.Errorf(" tcp and rtu are mutually exclusive.")
	}

	if pcc.Com.SerialPort != "" {
		modbusRtu := globals.ModbusRtu{
			SlaveId:    byte(md.Instance.PProtocol.ProtocolConfigs.SlaveID),
			SerialName: pcc.Com.SerialPort,
			BaudRate:   int(pcc.Com.BaudRate),
			DataBits:   int(pcc.Com.DataBits),
			StopBits:   int(pcc.Com.StopBits),
			Parity:     pcc.Com.Parity}
		md.ModbusClient, _ = dev.NewClient(modbusRtu)
	} else if pcc.Tcp.Ip != "" {
		modbusTcp := globals.ModbusTcp{
			SlaveId:  byte(md.Instance.PProtocol.ProtocolConfigs.SlaveID),
			DeviceIp: pcc.Tcp.Ip,
			TcpPort:  strconv.FormatInt(pcc.Tcp.Port, 10)}
		md.ModbusClient, _ = dev.NewClient(modbusTcp)
	}

	// set expected value
	for i := 0; i < len(md.Instance.Twins); i++ {
		var mvc configmap.ModbusVisitorConfig
		if err := json.Unmarshal([]byte(md.Instance.Twins[i].PVisitor.VisitorConfig), &mvc); err != nil {
			klog.Error(err)
			continue
		}
		// only ReadWrite AccessMode need
		if md.Instance.Twins[i].PVisitor.PProperty.AccessMode == "ReadWrite" {
			v, err := mappercommon.Convert(md.Instance.Twins[i].PVisitor.PProperty.DataType, md.Instance.Twins[i].Desired.Value)
			if err != nil {
				klog.Error("Convert error. Type: %s, value: %s ", md.Instance.Twins[i].PVisitor.PProperty.DataType, md.Instance.Twins[i].Desired.Value)
				continue
			}
			vint, ok := v.(int64)
			if !ok {
				klog.Error("convert type error")
				continue
			}
			_, err = dev.Set(md.ModbusClient, mvc.Register, mvc.Offset, uint16(vint))
			if err != nil {
				klog.Errorf("set expected value error: %v", err)
				continue
			}
		}
		td := TwinData{
			Client:       md.ModbusClient,
			Name:         md.Instance.Twins[i].PropertyName,
			Type:         md.Instance.Twins[i].Desired.Metadatas.Type,
			RegisterType: mvc.Register,
			Address:      mvc.Offset,
			Quantity:     uint16(mvc.Limit),
			Topic:        fmt.Sprintf(mappercommon.TopicTwinUpdate, md.Instance.ID)}
		c := time.Duration(md.Instance.Twins[i].PVisitor.CollectCycle)
		// set default CollectCycle if CollectCycle is null
		if c == 0 {
			c = 1 * time.Second
		}
		t := mappercommon.Timer{td.Run, c, 0}
		wg.Add(1)
		defer wg.Done()
		go t.Start()
	}
	// timer get data
	for i := 0; i < len(md.Instance.Datas.Properties); i++ {
		var mvc configmap.ModbusVisitorConfig
		if err := json.Unmarshal([]byte(md.Instance.Datas.Properties[i].PVisitor.VisitorConfig), &mvc); err != nil {
			klog.Error("Unmarshal visitor config failed")
			continue
		}
		td := TwinData{
			Client:       md.ModbusClient,
			Name:         md.Instance.Datas.Properties[i].PropertyName,
			Type:         md.Instance.Datas.Properties[i].Metadatas.Type,
			RegisterType: mvc.Register,
			Address:      mvc.Offset,
			Quantity:     uint16(mvc.Limit),
			Topic:        md.Instance.Datas.Topic}
		t := mappercommon.Timer{td.Run, time.Duration(md.Instance.Twins[i].PVisitor.CollectCycle), 0}
		c := time.Duration(md.Instance.Twins[i].PVisitor.CollectCycle)
		// set default CollectCycle if CollectCycle is null
		if c == 0 {
			c = 1 * time.Second
		}
		wg.Add(1)
		defer wg.Done()
		go t.Start()
	}
	//Subscribe the TwinUpdate topic
	topic := fmt.Sprintf(mappercommon.TopicTwinUpdateDelta, md.Instance.ID)
	if err = globals.MqttClient.Subscribe(topic, onMessage); err != nil {
		klog.Error(err)
		return
	}
	wg.Wait()
	return
	/*
		// timer get status and send to eventbus
		gs := GetSt{Client: md.ModbusClient,
			topic: fmt.Sprintf(mappercommon.TopicStateUpdate, md.Instance.ID)}
		t := mappercommon.Timer{gs.Run, 1 * time.Second, 0}
		go t.Start()
		wg.Add(1)
	*/
}
