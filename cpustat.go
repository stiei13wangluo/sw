package sw

import (
	"errors"
	"log"
	"strings"
	"time"

	"github.com/ulricqin/gosnmp"
)

func CpuUtilization(ip, community string, timeout, retry int) (int, error) {
	defer func() {
		if r := recover(); r != nil {
			log.Println(ip+" Recovered in CPUtilization", r)
		}
	}()
	vendor, err := SysVendor(ip, community, retry, timeout)
	if err != nil {
		return 0, err
	}
	method := "get"
	var oid string

	switch vendor {
	case "Cisco_NX":
		oid = "1.3.6.1.4.1.9.9.305.1.1.1.0"
	case "Cisco", "Cisco_IOS_7200", "Cisco_old":
		oid = "1.3.6.1.4.1.9.9.109.1.1.1.1.7.1"
	case "Hillstone":
		oid = ".1.3.6.1.4.1.28557.2.2.1.3"
		method = "getnext"
	case "Cisco_IOS_XE", "Cisco_IOS_XR":
		oid = "1.3.6.1.4.1.9.9.109.1.1.1.1.7"
		method = "getnext"
	case "Cisco_ASA":
		oid = "1.3.6.1.4.1.9.9.109.1.1.1.1.7"
		return getCiscoASAcpu(ip, community, oid, timeout, retry)
	case "Cisco_ASA_OLD":
		oid = "1.3.6.1.4.1.9.9.109.1.1.1.1.4"
		return getCiscoASAcpu(ip, community, oid, timeout, retry)
	case "Huawei", "Huawei_V5.70", "Huawei_V5.130":
		oid = "1.3.6.1.4.1.2011.5.25.31.1.1.1.1.5"
		return getH3CHWcpumem(ip, community, oid, timeout, retry)
	case "Huawei_V3.10", "H3c_V3.10":
		oid = "1.3.6.1.4.1.2011.6.1.1.1.3"
		return getH3CHWcpumem(ip, community, oid, timeout, retry)
	case "Huawei_ME60":
		oid = "1.3.6.1.4.1.2011.6.3.4.1.2"
		return getHuawei_ME60cpu(ip, community, oid, timeout, retry)
	case "H3C", "H3C_V5", "H3C_V7":
		oid = "1.3.6.1.4.1.25506.2.6.1.1.1.1.6"
		return getH3CHWcpumem(ip, community, oid, timeout, retry)
	case "H3C_S9500":
		oid = "1.3.6.1.4.1.2011.10.2.6.1.1.1.1.6"
		return getH3CHWcpumem(ip, community, oid, timeout, retry)
	case "Juniper":
		oid = "1.3.6.1.4.1.2636.3.1.13.1.8"
		return getH3CHWcpumem(ip, community, oid, timeout, retry)
	case "Ruijie":
		oid = "1.3.6.1.4.1.4881.1.1.10.2.36.1.1.2"
		return getRuijiecpumem(ip, community, oid, timeout, retry)
	case "Dell":
		oid = "1.3.6.1.4.1.674.10895.5000.2.6132.1.1.1.1.4.11"
		return getSnmpNextCpu(ip, community, oid, timeout, retry)
	case "Linux":
		oid = "1.3.6.1.4.1.2021.11.11.0"
		return getLinuxCpu(ip, community, oid, timeout, retry)
	case "A10":
		oid = "1.3.6.1.4.1.22610.2.4.1.3.3"
		return getSnmpNextCpu(ip, community, oid, timeout, retry)
	case "Aruba":
		oid = "1.3.6.1.4.1.14823.2.2.1.1.1.9.1.3"
		return getSnmpNextCpu(ip, community, oid, timeout, retry)
	case "Cisco_Controller":
		oid = "1.3.6.1.4.1.14179.1.1.5.1"
		return getSnmpNextCpu(ip, community, oid, timeout, retry)
	case "ZXCTN_9000":
		oid = "1.3.6.1.4.1.3902.3.6002.2.1.1.7"
		return GetZteCpuMem(ip, community, oid, timeout, retry)
	default:
		err = errors.New(ip + " Switch Vendor is not defined")
		return 0, err
	}

	var snmpPDUs []gosnmp.SnmpPDU
	for i := 0; i < retry; i++ {
		snmpPDUs, err = RunSnmp(ip, community, oid, method, timeout)
		if len(snmpPDUs) > 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if err == nil {
		for _, pdu := range snmpPDUs {
			return pdu.Value.(int), err
		}
	}

	return 0, err
}

func getCiscoASAcpu(ip, community, oid string, timeout, retry int) (value int, err error) {
	CPU_Value_SUM, CPU_Count, err := snmp_walk_sum(ip, community, oid, timeout, retry)
	if err == nil {
		if CPU_Count > 0 {
			return int(CPU_Value_SUM / CPU_Count), err
		}
	}
	return 0, err
}

func getH3CHWcpumem(ip, community, oid string, timeout, retry int) (value int, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Println(ip+" Recovered in CPUtilization", r)
		}
	}()
	method := "getnext"
	oidnext := oid
	var snmpPDUs []gosnmp.SnmpPDU

	for {
		for i := 0; i < retry; i++ {
			snmpPDUs, err = RunSnmp(ip, community, oidnext, method, timeout)
			if len(snmpPDUs) > 0 {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		oidnext = snmpPDUs[0].Name
		if strings.Contains(oidnext, oid) {
			if snmpPDUs[0].Value.(int) != 0 {
				value = snmpPDUs[0].Value.(int)
				break
			}
		} else {
			break
		}

	}

	return value, err
}

func GetZteCpuMem(ip, community, oid string, timeout, retry int) (value int, err error) {

	defer func() {
		if r := recover(); r != nil {
			log.Println(ip+" Recovered in CPUtilization", r)
		}
	}()

	//获取主引擎索性
	method := "get"
	masterOidList := []string{"1.3.6.1.4.1.3902.3.6002.1.3.1.6.0.0.3", "1.3.6.1.4.1.3902.3.6002.1.3.1.6.0.0.4"}
	var snmpPDUs []gosnmp.SnmpPDU
	masterIndex := ""
	for _, masterOid := range masterOidList {
		for i := 0; i < retry; i++ {
			snmpPDUs, err = RunSnmp(ip, community, masterOid, method, timeout)
			if len(snmpPDUs) > 0 {
				if snmpPDUs[0].Value.(int) == 0 {
					runes := []rune(snmpPDUs[0].Name)
					masterIndex = string(runes[len(runes)-1])
				}
				break
			}
			time.Sleep(100 * time.Millisecond)
		}

		if masterIndex != "" {
			break
		}
	}
	oid = oid + ".0.0." + masterIndex + ".0"
	for i := 0; i < retry; i++ {
		snmpPDUs, err = RunSnmp(ip, community, oid, method, timeout)
		if len(snmpPDUs) > 0 {
			value = snmpPDUs[0].Value.(int)
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	return value, err
}

func getRuijiecpumem(ip, community, oid string, timeout, retry int) (value int, err error) {

	defer func() {
		if r := recover(); r != nil {
			log.Println(ip+" Recovered in CPUtilization", r)
		}
	}()
	method := "getnext"

	var snmpPDUs []gosnmp.SnmpPDU

	for i := 0; i < retry; i++ {
		snmpPDUs, err = RunSnmp(ip, community, oid, method, timeout)
		if len(snmpPDUs) > 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	return snmpPDUs[0].Value.(int), err
}

func getHuawei_ME60cpu(ip, community, oid string, timeout, retry int) (value int, err error) {
	CPU_Value_SUM, CPU_Count, err := snmp_walk_sum(ip, community, oid, timeout, retry)
	if err == nil {
		if CPU_Count > 0 {
			return int(CPU_Value_SUM / CPU_Count), err
		}
	}

	return 0, err
}

func getSnmpNextCpu(ip, community, oid string, timeout, retry int) (value int, err error) {

	defer func() {
		if r := recover(); r != nil {
			log.Println(ip+" Recovered in CPUtilization", r)
		}
	}()
	method := "getnext"

	var snmpPDUs []gosnmp.SnmpPDU

	for i := 0; i < retry; i++ {
		snmpPDUs, err = RunSnmp(ip, community, oid, method, timeout)
		if len(snmpPDUs) > 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	return snmpPDUs[0].Value.(int), err
}

func getLinuxCpu(ip, community, oid string, timeout, retry int) (value int, err error) {

	defer func() {
		if r := recover(); r != nil {
			log.Println(ip+" Recovered in CPUtilization", r)
		}
	}()
	method := "get"

	var snmpPDUs []gosnmp.SnmpPDU

	for i := 0; i < retry; i++ {
		snmpPDUs, err = RunSnmp(ip, community, oid, method, timeout)
		if len(snmpPDUs) > 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	return (100 - snmpPDUs[0].Value.(int)), err
}

func snmp_walk_sum(ip, community, oid string, timeout, retry int) (value_sum int, value_count int, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Println(ip+" Recovered in CPUtilization", r)
		}
	}()
	var snmpPDUs []gosnmp.SnmpPDU
	method := "walk"
	for i := 0; i < retry; i++ {
		snmpPDUs, err = RunSnmp(ip, community, oid, method, timeout)
		if len(snmpPDUs) > 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	var Values []int
	if err == nil {
		for _, pdu := range snmpPDUs {
			Values = append(Values, pdu.Value.(int))
		}
	}
	var Value_SUM int
	Value_SUM = 0
	for _, value := range Values {
		Value_SUM = Value_SUM + value
	}
	return Value_SUM, len(Values), err
}
