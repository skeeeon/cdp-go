package cdp

import "fmt"

// init builds the type registry. One entry per supported data item type ID;
// each entry knows the Go name, the NATS subject token (lowercase, version
// suffix stripped), and the decoder.
func init() {
	registry[0x010D] = registryEntry{"NodeStatusChangeV2", "node_status_change", decodeNodeStatusChangeV2}
	registry[0x011A] = registryEntry{"CDPStreamInformation", "cdp_stream_information", decodeCDPStreamInformation}
	registry[0x011B] = registryEntry{"HostnameAnnounce", "hostname_announce", decodeHostnameAnnounce}
	registry[0x011C] = registryEntry{"InstanceAnnounce", "instance_announce", decodeInstanceAnnounce}
	registry[0x0121] = registryEntry{"AppSettingsChunk", "app_settings_chunk", decodeAppSettingsChunk}
	registry[0x0127] = registryEntry{"DistanceV2", "distance", decodeDistanceV2}
	registry[0x0135] = registryEntry{"PositionV3", "position", decodePositionV3}
	registry[0x0136] = registryEntry{"AnchorPositionStatusV3", "anchor_position_status", decodeAnchorPositionStatusV3}
	registry[0x0137] = registryEntry{"DeviceActivityState", "device_activity_state", decodeDeviceActivityState}
	registry[0x0138] = registryEntry{"DeviceHardwareStatusV2", "device_hardware_status", decodeDeviceHardwareStatusV2}
	registry[0x0139] = registryEntry{"AccelerometerV2", "accelerometer", decodeAccelerometerV2}
	registry[0x013A] = registryEntry{"GyroscopeV2", "gyroscope", decodeGyroscopeV2}
	registry[0x013B] = registryEntry{"MagnetometerV2", "magnetometer", decodeMagnetometerV2}
	registry[0x013C] = registryEntry{"PressureV2", "pressure", decodePressureV2}
	registry[0x013D] = registryEntry{"QuaternionV2", "quaternion", decodeQuaternionV2}
	registry[0x013E] = registryEntry{"TemperatureV2", "temperature", decodeTemperatureV2}
	registry[0x013F] = registryEntry{"DeviceNames", "device_names", decodeDeviceNames}
	registry[0x0140] = registryEntry{"Synchronization", "synchronization", decodeSynchronization}
	registry[0x0141] = registryEntry{"RoleReport", "role_report", decodeRoleReport}
	registry[0x0148] = registryEntry{"UserDefinedV2", "user_defined", decodeUserDefinedV2}
	registry[0x0149] = registryEntry{"NetworkTime", "network_time", decodeNetworkTime}
	registry[0x014A] = registryEntry{"AnchorHealthV5", "anchor_health", decodeAnchorHealthV5}
	registry[0x014C] = registryEntry{"GlobalPingTimingReportV1", "global_ping_timing_report", decodeGlobalPingTimingReportV1}
	registry[0x015A] = registryEntry{"NtRealTimeMappingV1", "nt_realtime_mapping", decodeNtRealTimeMappingV1}
	registry[0x0160] = registryEntry{"BootloadProgress", "bootload_progress", decodeBootloadProgress}
	registry[0x0161] = registryEntry{"AnchorPositionStatusV4", "anchor_position_status", decodeAnchorPositionStatusV4}
	registry[0x0163] = registryEntry{"LinkMDStatus", "link_md_status", decodeLinkMDStatus}
	registry[0x0164] = registryEntry{"PolarCoordinatesV1", "polar_coordinates", decodePolarCoordinatesV1}
	registry[0x016A] = registryEntry{"BoundingBoxReport", "bounding_box_report", decodeBoundingBoxReport}
	registry[0x016B] = registryEntry{"BoundingCylinderReport", "bounding_cylinder_report", decodeBoundingCylinderReport}
	registry[0x0171] = registryEntry{"ImageDiscoveryV2", "image_discovery", decodeImageDiscoveryV2}
	registry[0x0178] = registryEntry{"QuaternionV3", "quaternion", decodeQuaternionV3}
	registry[0x0179] = registryEntry{"UserDefinedV3", "user_defined", decodeUserDefinedV3}
	registry[0x017A] = registryEntry{"AccelerometerV3", "accelerometer", decodeAccelerometerV3}
	registry[0x017B] = registryEntry{"GyroscopeV3", "gyroscope", decodeGyroscopeV3}
	registry[0x017C] = registryEntry{"MagnetometerV3", "magnetometer", decodeMagnetometerV3}
	registry[0x8009] = registryEntry{"ImageDiscoveryV1", "image_discovery", decodeImageDiscoveryV1}
	registry[0x802C] = registryEntry{"TimedRxV5", "timed_rx", decodeTimedRxV5}
	registry[0x802D] = registryEntry{"TickV4", "tick", decodeTickV4}
	registry[0x802F] = registryEntry{"PingV5", "ping", decodePingV5}
	registry[0x80B2] = registryEntry{"TickV5", "tick", decodeTickV5}
	registry[0x80B3] = registryEntry{"TimedRxV6", "timed_rx", decodeTimedRxV6}
	registry[0x80C0] = registryEntry{"DeviceColor", "device_color", decodeDeviceColor}
	registry[0x80D4] = registryEntry{"DeviceStatusV3", "device_status", decodeDeviceStatusV3}
	registry[0x80DA] = registryEntry{"ClearDeviceColor", "clear_device_color", decodeClearDeviceColor}
	registry[0x80DB] = registryEntry{"GeofencerZoneInfo", "geofencer_zone_info", decodeGeofencerZoneInfo}
	registry[0x80DC] = registryEntry{"TagZoneInfo", "tag_zone_info", decodeTagZoneInfo}
	registry[0x80DD] = registryEntry{"DrawPrism", "draw_prism", decodeDrawPrism}
	registry[0x80DE] = registryEntry{"ClearObject", "clear_object", decodeClearObject}

	// Precompute the "0xNNNN" string form once per registered type so
	// per-item publishers can read it off Item.TypeHex without an Sprintf
	// on the hot path.
	for tid := range registry {
		typeHexes[tid] = fmt.Sprintf("0x%04X", tid)
	}
}

// ---- Shared sub-structures ------------------------------------------------

// FullDeviceID is the (serial, interface_id) pair used inside several items.
type FullDeviceID struct {
	SerialNumber Serial `json:"serial_number"`
	InterfaceID  uint8  `json:"interface_id"`
}

// ErrorPattern is one byte encoding three 2-bit LED color codes.
type ErrorPattern struct {
	Pattern uint8 `json:"pattern"`
}

// PositionAnchorStatus is one entry in AnchorPositionStatusV3.AnchorStatusArray.
type PositionAnchorStatus struct {
	AnchorSerialNumber  Serial `json:"anchor_serial_number"`
	AnchorInterfaceID   uint8  `json:"anchor_interface_id"`
	Status              uint8  `json:"status"`
	FirstPath           int16  `json:"first_path"`
	TotalPath           int16  `json:"total_path"`
	Quality             uint16 `json:"quality"`
}

// PositionAnchorStatusV4Entry is one entry in AnchorPositionStatusV4.AnchorStatusArray.
type PositionAnchorStatusV4Entry struct {
	AnchorSerialNumber Serial `json:"anchor_serial_number"`
	AnchorInterfaceID  uint8  `json:"anchor_interface_id"`
	Status             uint8  `json:"status"`
	Quality            uint16 `json:"quality"`
}

// XYCoordinate is an (x, y) vertex used in geofencing/draw items, in millimeters.
type XYCoordinate struct {
	X int32 `json:"x"`
	Y int32 `json:"y"`
}

// UWBSignalStrength is the 12-byte signal-strength block embedded in
// TimedRxV5/V6 and PingV5. Six little-endian uint16 fields, in order.
type UWBSignalStrength struct {
	FpAmpl1       uint16 `json:"fp_ampl1"`
	FpAmpl2       uint16 `json:"fp_ampl2"`
	FpAmpl3       uint16 `json:"fp_ampl3"`
	RxPreambleAcc uint16 `json:"rx_preamble_acc"`
	CIRPower      uint16 `json:"cir_power"`
	StdNoise      uint16 `json:"std_noise"`
}

// Image is one entry in ImageDiscoveryV1.ImageInformation; 53 bytes wide.
type Image struct {
	Type    uint8  `json:"type"`
	Version string `json:"version"`
	SHA1    []byte `json:"sha1"`
}

// decodeSignalStrength decodes a 12-byte UWBSignalStrength block at offset off.
// The caller must ensure b has at least off+12 bytes.
func decodeSignalStrength(b []byte, off int) UWBSignalStrength {
	return UWBSignalStrength{
		FpAmpl1:       u16(b, off),
		FpAmpl2:       u16(b, off+2),
		FpAmpl3:       u16(b, off+4),
		RxPreambleAcc: u16(b, off+6),
		CIRPower:      u16(b, off+8),
		StdNoise:      u16(b, off+10),
	}
}

// ---- 0x0135 PositionV3 ----------------------------------------------------

type PositionV3 struct {
	SerialNumber Serial `json:"serial_number"`
	NetworkTime  uint64 `json:"network_time"`
	X            int32  `json:"x"`
	Y            int32  `json:"y"`
	Z            int32  `json:"z"`
	Quality      uint16 `json:"quality"`
	AnchorCount  uint8  `json:"anchor_count"`
	Flags        uint8  `json:"flags"`
	Smoothing    uint16 `json:"smoothing"`
}

func decodePositionV3(b []byte) (any, error) {
	if len(b) < 30 {
		return nil, errShort
	}
	return &PositionV3{
		SerialNumber: Serial(u32(b, 0)),
		NetworkTime:  u64(b, 4),
		X:            i32(b, 12),
		Y:            i32(b, 16),
		Z:            i32(b, 20),
		Quality:      u16(b, 24),
		AnchorCount:  u8(b, 26),
		Flags:        u8(b, 27),
		Smoothing:    u16(b, 28),
	}, nil
}

// ---- 0x0136 AnchorPositionStatusV3 ----------------------------------------

type AnchorPositionStatusV3 struct {
	TagSerialNumber   Serial                 `json:"tag_serial_number"`
	NetworkTime       uint64                 `json:"network_time"`
	AnchorStatusArray []PositionAnchorStatus `json:"anchor_status_array"`
}

func decodeAnchorPositionStatusV3(b []byte) (any, error) {
	if len(b) < 12 {
		return nil, errShort
	}
	out := &AnchorPositionStatusV3{
		TagSerialNumber: Serial(u32(b, 0)),
		NetworkTime:     u64(b, 4),
	}
	rest := b[12:]
	const itemSize = 12 // serial(4)+iface(1)+status(1)+fp(2)+tp(2)+q(2)
	for len(rest) >= itemSize {
		out.AnchorStatusArray = append(out.AnchorStatusArray, PositionAnchorStatus{
			AnchorSerialNumber: Serial(u32(rest, 0)),
			AnchorInterfaceID:  u8(rest, 4),
			Status:             u8(rest, 5),
			FirstPath:          i16(rest, 6),
			TotalPath:          i16(rest, 8),
			Quality:            u16(rest, 10),
		})
		rest = rest[itemSize:]
	}
	if len(rest) != 0 {
		return nil, fmt.Errorf("anchor_position_status_v3: trailing %d bytes", len(rest))
	}
	return out, nil
}

// ---- 0x0137 DeviceActivityState -------------------------------------------

type DeviceActivityState struct {
	SerialNumber          Serial `json:"serial_number"`
	InterfaceID           uint8  `json:"interface_id"`
	X                     int32  `json:"x"`
	Y                     int32  `json:"y"`
	Z                     int32  `json:"z"`
	RoleID                uint8  `json:"role_id"`
	ConnectivityState     uint8  `json:"connectivity_state"`
	SynchronizationState  uint8  `json:"synchronization_state"`
}

func decodeDeviceActivityState(b []byte) (any, error) {
	if len(b) < 20 {
		return nil, errShort
	}
	return &DeviceActivityState{
		SerialNumber:         Serial(u32(b, 0)),
		InterfaceID:          u8(b, 4),
		X:                    i32(b, 5),
		Y:                    i32(b, 9),
		Z:                    i32(b, 13),
		RoleID:               u8(b, 17),
		ConnectivityState:    u8(b, 18),
		SynchronizationState: u8(b, 19),
	}, nil
}

// ---- 0x0138 DeviceHardwareStatusV2 ----------------------------------------

type DeviceHardwareStatusV2 struct {
	SerialNumber      Serial         `json:"serial_number"`
	Memory            uint32         `json:"memory"`
	Flags             uint32         `json:"flags"`
	MinutesRemaining  uint16         `json:"minutes_remaining"`
	BatteryPercentage uint8          `json:"battery_percentage"`
	Temperature       int8           `json:"temperature"`
	ProcessorUsage    uint8          `json:"processor_usage"`
	ErrorPatterns     []ErrorPattern `json:"error_patterns"`
}

func decodeDeviceHardwareStatusV2(b []byte) (any, error) {
	if len(b) < 17 {
		return nil, errShort
	}
	out := &DeviceHardwareStatusV2{
		SerialNumber:      Serial(u32(b, 0)),
		Memory:            u32(b, 4),
		Flags:             u32(b, 8),
		MinutesRemaining:  u16(b, 12),
		BatteryPercentage: u8(b, 14),
		Temperature:       i8(b, 15),
		ProcessorUsage:    u8(b, 16),
	}
	for _, p := range b[17:] {
		out.ErrorPatterns = append(out.ErrorPatterns, ErrorPattern{Pattern: p})
	}
	return out, nil
}

// ---- 0x0139 AccelerometerV2 -----------------------------------------------

type AccelerometerV2 struct {
	SerialNumber Serial `json:"serial_number"`
	NetworkTime  uint64 `json:"network_time"`
	X            int32  `json:"x"`
	Y            int32  `json:"y"`
	Z            int32  `json:"z"`
	Scale        uint8  `json:"scale"`
}

func decodeAccelerometerV2(b []byte) (any, error) {
	if len(b) < 25 {
		return nil, errShort
	}
	return &AccelerometerV2{
		SerialNumber: Serial(u32(b, 0)),
		NetworkTime:  u64(b, 4),
		X:            i32(b, 12),
		Y:            i32(b, 16),
		Z:            i32(b, 20),
		Scale:        u8(b, 24),
	}, nil
}

// ---- 0x013A GyroscopeV2 ---------------------------------------------------

type GyroscopeV2 struct {
	SerialNumber Serial `json:"serial_number"`
	NetworkTime  uint64 `json:"network_time"`
	X            int32  `json:"x"`
	Y            int32  `json:"y"`
	Z            int32  `json:"z"`
	Scale        uint16 `json:"scale"`
}

func decodeGyroscopeV2(b []byte) (any, error) {
	if len(b) < 26 {
		return nil, errShort
	}
	return &GyroscopeV2{
		SerialNumber: Serial(u32(b, 0)),
		NetworkTime:  u64(b, 4),
		X:            i32(b, 12),
		Y:            i32(b, 16),
		Z:            i32(b, 20),
		Scale:        u16(b, 24),
	}, nil
}

// ---- 0x013B MagnetometerV2 ------------------------------------------------

type MagnetometerV2 struct {
	SerialNumber Serial `json:"serial_number"`
	NetworkTime  uint64 `json:"network_time"`
	X            int32  `json:"x"`
	Y            int32  `json:"y"`
	Z            int32  `json:"z"`
	Scale        uint16 `json:"scale"`
}

func decodeMagnetometerV2(b []byte) (any, error) {
	if len(b) < 26 {
		return nil, errShort
	}
	return &MagnetometerV2{
		SerialNumber: Serial(u32(b, 0)),
		NetworkTime:  u64(b, 4),
		X:            i32(b, 12),
		Y:            i32(b, 16),
		Z:            i32(b, 20),
		Scale:        u16(b, 24),
	}, nil
}

// ---- 0x013C PressureV2 ----------------------------------------------------

type PressureV2 struct {
	SerialNumber Serial `json:"serial_number"`
	NetworkTime  uint64 `json:"network_time"`
	Pressure     int32  `json:"pressure"`
	Scale        uint32 `json:"scale"`
}

func decodePressureV2(b []byte) (any, error) {
	if len(b) < 20 {
		return nil, errShort
	}
	return &PressureV2{
		SerialNumber: Serial(u32(b, 0)),
		NetworkTime:  u64(b, 4),
		Pressure:     i32(b, 12),
		Scale:        u32(b, 16),
	}, nil
}

// ---- 0x013D QuaternionV2 --------------------------------------------------

type QuaternionV2 struct {
	SerialNumber   Serial `json:"serial_number"`
	NetworkTime    uint64 `json:"network_time"`
	X              int32  `json:"x"`
	Y              int32  `json:"y"`
	Z              int32  `json:"z"`
	W              int32  `json:"w"`
	QuaternionType uint8  `json:"quaternion_type"`
}

func decodeQuaternionV2(b []byte) (any, error) {
	if len(b) < 29 {
		return nil, errShort
	}
	return &QuaternionV2{
		SerialNumber:   Serial(u32(b, 0)),
		NetworkTime:    u64(b, 4),
		X:              i32(b, 12),
		Y:              i32(b, 16),
		Z:              i32(b, 20),
		W:              i32(b, 24),
		QuaternionType: u8(b, 28),
	}, nil
}

// ---- 0x013E TemperatureV2 -------------------------------------------------

type TemperatureV2 struct {
	SerialNumber Serial `json:"serial_number"`
	NetworkTime  uint64 `json:"network_time"`
	Temperature  int16  `json:"temperature"`
	Scale        uint16 `json:"scale"`
}

func decodeTemperatureV2(b []byte) (any, error) {
	if len(b) < 16 {
		return nil, errShort
	}
	return &TemperatureV2{
		SerialNumber: Serial(u32(b, 0)),
		NetworkTime:  u64(b, 4),
		Temperature:  i16(b, 12),
		Scale:        u16(b, 14),
	}, nil
}

// ---- 0x013F DeviceNames ---------------------------------------------------

type DeviceNames struct {
	SerialNumber Serial `json:"serial_number"`
	Name         string `json:"name"`
}

func decodeDeviceNames(b []byte) (any, error) {
	if len(b) < 4 {
		return nil, errShort
	}
	return &DeviceNames{
		SerialNumber: Serial(u32(b, 0)),
		Name:         nullStrip(b[4:]),
	}, nil
}

// ---- 0x0140 Synchronization -----------------------------------------------

type Synchronization struct {
	MaxTxSyncCount     uint16 `json:"max_tx_sync_count"`
	CurrentTxSyncCount uint16 `json:"current_tx_sync_count"`
	MaxRxSyncCount     uint16 `json:"max_rx_sync_count"`
	CurrentRxSyncCount uint16 `json:"current_rx_sync_count"`
}

func decodeSynchronization(b []byte) (any, error) {
	if len(b) < 8 {
		return nil, errShort
	}
	return &Synchronization{
		MaxTxSyncCount:     u16(b, 0),
		CurrentTxSyncCount: u16(b, 2),
		MaxRxSyncCount:     u16(b, 4),
		CurrentRxSyncCount: u16(b, 6),
	}, nil
}

// ---- 0x0141 RoleReport ----------------------------------------------------

type RoleReport struct {
	RoleID         uint16 `json:"role_id"`
	MaxQuantity    uint16 `json:"max_quantity"`
	ActiveQuantity uint16 `json:"active_quantity"`
	RoleName       string `json:"role_name"`
}

func decodeRoleReport(b []byte) (any, error) {
	if len(b) < 6 {
		return nil, errShort
	}
	return &RoleReport{
		RoleID:         u16(b, 0),
		MaxQuantity:    u16(b, 2),
		ActiveQuantity: u16(b, 4),
		RoleName:       nullStrip(b[6:]),
	}, nil
}

// ---- 0x0148 UserDefinedV2 -------------------------------------------------

type UserDefinedV2 struct {
	SerialNumber Serial `json:"serial_number"`
	Payload      []byte `json:"payload"`
}

func decodeUserDefinedV2(b []byte) (any, error) {
	if len(b) < 4 {
		return nil, errShort
	}
	payload := append([]byte(nil), b[4:]...)
	return &UserDefinedV2{
		SerialNumber: Serial(u32(b, 0)),
		Payload:      payload,
	}, nil
}

// ---- 0x0149 NetworkTime ---------------------------------------------------

type NetworkTime struct {
	ServerInstance Serial `json:"server_instance"`
	NetworkTime    uint64 `json:"network_time"`
	NTQuality      uint8  `json:"nt_quality"`
}

func decodeNetworkTime(b []byte) (any, error) {
	if len(b) < 13 {
		return nil, errShort
	}
	return &NetworkTime{
		ServerInstance: Serial(u32(b, 0)),
		NetworkTime:    u64(b, 4),
		NTQuality:      u8(b, 12),
	}, nil
}

// ---- 0x014A AnchorHealthV5 ------------------------------------------------

type AnchorHealthV5 struct {
	SerialNumber              Serial         `json:"serial_number"`
	InterfaceID               uint8          `json:"interface_id"`
	TicksReported             uint32         `json:"ticks_reported"`
	TimedRxsReported          uint32         `json:"timed_rxs_reported"`
	BeaconsReported           uint32         `json:"beacons_reported"`
	BeaconsDiscarded          uint32         `json:"beacons_discarded"`
	BeaconsLate               uint32         `json:"beacons_late"`
	AverageQuality            uint16         `json:"average_quality"`
	ReportPeriod              uint8          `json:"report_period"`
	InteranchorCommsErrorCode uint8          `json:"interanchor_comms_error_code"`
	BadPairedAnchors          []FullDeviceID `json:"bad_paired_anchors"`
}

func decodeAnchorHealthV5(b []byte) (any, error) {
	if len(b) < 29 {
		return nil, errShort
	}
	out := &AnchorHealthV5{
		SerialNumber:              Serial(u32(b, 0)),
		InterfaceID:               u8(b, 4),
		TicksReported:             u32(b, 5),
		TimedRxsReported:          u32(b, 9),
		BeaconsReported:           u32(b, 13),
		BeaconsDiscarded:          u32(b, 17),
		BeaconsLate:               u32(b, 21),
		AverageQuality:            u16(b, 25),
		ReportPeriod:              u8(b, 27),
		InteranchorCommsErrorCode: u8(b, 28),
	}
	rest := b[29:]
	const stride = 5
	for len(rest) >= stride {
		out.BadPairedAnchors = append(out.BadPairedAnchors, FullDeviceID{
			SerialNumber: Serial(u32(rest, 0)),
			InterfaceID:  u8(rest, 4),
		})
		rest = rest[stride:]
	}
	if len(rest) != 0 {
		return nil, fmt.Errorf("anchor_health_v5: trailing %d bytes", len(rest))
	}
	return out, nil
}

// ---- 0x015A NtRealTimeMappingV1 -------------------------------------------

type NtRealTimeMappingV1 struct {
	NetworkTimePrevious uint64 `json:"network_time_previous"`
	RealTimePrevious    uint64 `json:"real_time_previous"`
	NetworkTimeCurrent  uint64 `json:"network_time_current"`
	RealTimeCurrent     uint64 `json:"real_time_current"`
}

func decodeNtRealTimeMappingV1(b []byte) (any, error) {
	if len(b) < 32 {
		return nil, errShort
	}
	return &NtRealTimeMappingV1{
		NetworkTimePrevious: u64(b, 0),
		RealTimePrevious:    u64(b, 8),
		NetworkTimeCurrent:  u64(b, 16),
		RealTimeCurrent:     u64(b, 24),
	}, nil
}

// ---- 0x0160 BootloadProgress ----------------------------------------------

type BootloadProgress struct {
	SerialNumber              Serial   `json:"serial_number"`
	LastReceivedTotalPathRSSI uint8    `json:"last_received_total_path_rssi"`
	LastHeardPacketTime       uint16   `json:"last_heard_packet_time"`
	Flags                     [25]byte `json:"flags"`
	MaxSectorsPerFlag         uint8    `json:"max_sectors_per_flag"`
	LastMaxSectorFlag         uint8    `json:"last_max_sector_flag"`
	Percentage                uint16   `json:"percentage"`
}

func decodeBootloadProgress(b []byte) (any, error) {
	if len(b) < 36 {
		return nil, errShort
	}
	out := &BootloadProgress{
		SerialNumber:              Serial(u32(b, 0)),
		LastReceivedTotalPathRSSI: u8(b, 4),
		LastHeardPacketTime:       u16(b, 5),
		MaxSectorsPerFlag:         u8(b, 32),
		LastMaxSectorFlag:         u8(b, 33),
		Percentage:                u16(b, 34),
	}
	copy(out.Flags[:], b[7:32])
	return out, nil
}

// ---- 0x0164 PolarCoordinatesV1 --------------------------------------------

type PolarCoordinatesV1 struct {
	SerialNumber Serial  `json:"serial_number"`
	NetworkTime  uint64  `json:"network_time"`
	Rho          uint32  `json:"rho"`
	Theta        float32 `json:"theta"`
	Phi          float32 `json:"phi"`
	Quality      uint16  `json:"quality"`
	AnchorCount  uint8   `json:"anchor_count"`
	Flags        uint8   `json:"flags"`
	Smoothing    uint16  `json:"smoothing"`
}

func decodePolarCoordinatesV1(b []byte) (any, error) {
	if len(b) < 30 {
		return nil, errShort
	}
	return &PolarCoordinatesV1{
		SerialNumber: Serial(u32(b, 0)),
		NetworkTime:  u64(b, 4),
		Rho:          u32(b, 12),
		Theta:        f32(b, 16),
		Phi:          f32(b, 20),
		Quality:      u16(b, 24),
		AnchorCount:  u8(b, 26),
		Flags:        u8(b, 27),
		Smoothing:    u16(b, 28),
	}, nil
}

// ---- 0x0171 ImageDiscoveryV2 ----------------------------------------------

type ImageDiscoveryV2 struct {
	VID               uint8  `json:"vid"`
	PID               uint8  `json:"pid"`
	RunningImageType  uint8  `json:"running_image_type"`
	TLVs              []byte `json:"tlvs"`
}

func decodeImageDiscoveryV2(b []byte) (any, error) {
	if len(b) < 3 {
		return nil, errShort
	}
	tlvs := append([]byte(nil), b[3:]...)
	return &ImageDiscoveryV2{
		VID:              u8(b, 0),
		PID:              u8(b, 1),
		RunningImageType: u8(b, 2),
		TLVs:             tlvs,
	}, nil
}

// ---- 0x010D NodeStatusChangeV2 --------------------------------------------

type NodeStatusChangeV2 struct {
	SerialNumber Serial `json:"serial_number"`
	InterfaceID  uint8  `json:"interface_id"`
	NodeStatus   uint8  `json:"node_status"`
}

func decodeNodeStatusChangeV2(b []byte) (any, error) {
	if len(b) < 6 {
		return nil, errShort
	}
	return &NodeStatusChangeV2{
		SerialNumber: Serial(u32(b, 0)),
		InterfaceID:  u8(b, 4),
		NodeStatus:   u8(b, 5),
	}, nil
}

// ---- 0x011A CDPStreamInformation ------------------------------------------

type CDPStreamInformation struct {
	DestinationIP    uint32 `json:"destination_ip"`
	DestinationPort  uint16 `json:"destination_port"`
	InterfaceIP      uint32 `json:"interface_ip"`
	InterfaceNetmask uint32 `json:"interface_netmask"`
	InterfacePort    uint16 `json:"interface_port"`
	TTL              uint8  `json:"ttl"`
	Name             string `json:"name"`
}

func decodeCDPStreamInformation(b []byte) (any, error) {
	if len(b) < 17 {
		return nil, errShort
	}
	return &CDPStreamInformation{
		DestinationIP:    u32(b, 0),
		DestinationPort:  u16(b, 4),
		InterfaceIP:      u32(b, 6),
		InterfaceNetmask: u32(b, 10),
		InterfacePort:    u16(b, 14),
		TTL:              u8(b, 16),
		Name:             nullStrip(b[17:]),
	}, nil
}

// ---- 0x011B HostnameAnnounce ----------------------------------------------

type HostnameAnnounce struct {
	Hostname string `json:"hostname"`
}

func decodeHostnameAnnounce(b []byte) (any, error) {
	return &HostnameAnnounce{Hostname: nullStrip(b)}, nil
}

// ---- 0x011C InstanceAnnounce ----------------------------------------------

type InstanceAnnounce struct {
	InstanceName string `json:"instance_name"`
}

func decodeInstanceAnnounce(b []byte) (any, error) {
	return &InstanceAnnounce{InstanceName: nullStrip(b)}, nil
}

// ---- 0x0121 AppSettingsChunk ----------------------------------------------

type AppSettingsChunk struct {
	NumberOfChunks uint16 `json:"number_of_chunks"`
	ChunkID        uint16 `json:"chunk_id"`
	InstanceName   string `json:"instance_name"`
	ChunkData      []byte `json:"chunk_data"`
}

func decodeAppSettingsChunk(b []byte) (any, error) {
	if len(b) < 260 {
		return nil, errShort
	}
	return &AppSettingsChunk{
		NumberOfChunks: u16(b, 0),
		ChunkID:        u16(b, 2),
		InstanceName:   nullStrip(b[4:260]),
		ChunkData:      append([]byte(nil), b[260:]...),
	}, nil
}

// ---- 0x0127 DistanceV2 ----------------------------------------------------

type DistanceV2 struct {
	SerialNumber1 Serial `json:"serial_number_1"`
	SerialNumber2 Serial `json:"serial_number_2"`
	InterfaceID1  uint8  `json:"interface_id_1"`
	InterfaceID2  uint8  `json:"interface_id_2"`
	RxTimestamp   uint64 `json:"rx_timestamp"`
	Distance      uint32 `json:"distance"`
	Quality       uint16 `json:"quality"`
}

func decodeDistanceV2(b []byte) (any, error) {
	if len(b) < 24 {
		return nil, errShort
	}
	return &DistanceV2{
		SerialNumber1: Serial(u32(b, 0)),
		SerialNumber2: Serial(u32(b, 4)),
		InterfaceID1:  u8(b, 8),
		InterfaceID2:  u8(b, 9),
		RxTimestamp:   u64(b, 10),
		Distance:      u32(b, 18),
		Quality:       u16(b, 22),
	}, nil
}

// ---- 0x014C GlobalPingTimingReportV1 --------------------------------------

type GlobalPingTimingReportV1 struct {
	InitialPingCount         uint32   `json:"initial_ping_count"`
	PositionCalculationDelay uint32   `json:"position_calculation_delay"`
	ArrivalTimeCounts        []uint32 `json:"arrival_time_counts"`
}

func decodeGlobalPingTimingReportV1(b []byte) (any, error) {
	if len(b) < 8 {
		return nil, errShort
	}
	rest := b[8:]
	if len(rest)%4 != 0 {
		return nil, fmt.Errorf("global_ping_timing_report_v1: trailing %d bytes", len(rest)%4)
	}
	counts := make([]uint32, len(rest)/4)
	for i := range counts {
		counts[i] = u32(rest, i*4)
	}
	return &GlobalPingTimingReportV1{
		InitialPingCount:         u32(b, 0),
		PositionCalculationDelay: u32(b, 4),
		ArrivalTimeCounts:        counts,
	}, nil
}

// ---- 0x0161 AnchorPositionStatusV4 ----------------------------------------

type AnchorPositionStatusV4 struct {
	TagSerialNumber   Serial                        `json:"tag_serial_number"`
	NetworkTime       uint64                        `json:"network_time"`
	AnchorStatusArray []PositionAnchorStatusV4Entry `json:"anchor_status_array"`
}

func decodeAnchorPositionStatusV4(b []byte) (any, error) {
	if len(b) < 12 {
		return nil, errShort
	}
	out := &AnchorPositionStatusV4{
		TagSerialNumber: Serial(u32(b, 0)),
		NetworkTime:     u64(b, 4),
	}
	rest := b[12:]
	const itemSize = 8 // serial(4)+iface(1)+status(1)+quality(2)
	for len(rest) >= itemSize {
		out.AnchorStatusArray = append(out.AnchorStatusArray, PositionAnchorStatusV4Entry{
			AnchorSerialNumber: Serial(u32(rest, 0)),
			AnchorInterfaceID:  u8(rest, 4),
			Status:             u8(rest, 5),
			Quality:            u16(rest, 6),
		})
		rest = rest[itemSize:]
	}
	if len(rest) != 0 {
		return nil, fmt.Errorf("anchor_position_status_v4: trailing %d bytes", len(rest))
	}
	return out, nil
}

// ---- 0x0163 LinkMDStatus --------------------------------------------------

type LinkMDStatus struct {
	PortNumber       uint8  `json:"port_number"`
	CableCondition   uint8  `json:"cable_condition"`
	DistanceToFault  uint32 `json:"distance_to_fault"`
}

func decodeLinkMDStatus(b []byte) (any, error) {
	if len(b) < 6 {
		return nil, errShort
	}
	return &LinkMDStatus{
		PortNumber:      u8(b, 0),
		CableCondition:  u8(b, 1),
		DistanceToFault: u32(b, 2),
	}, nil
}

// ---- 0x016A BoundingBoxReport ---------------------------------------------

type BoundingBoxReport struct {
	MinX uint32 `json:"min_x"`
	MinY uint32 `json:"min_y"`
	MinZ uint32 `json:"min_z"`
	MaxX uint32 `json:"max_x"`
	MaxY uint32 `json:"max_y"`
	MaxZ uint32 `json:"max_z"`
}

func decodeBoundingBoxReport(b []byte) (any, error) {
	if len(b) < 24 {
		return nil, errShort
	}
	return &BoundingBoxReport{
		MinX: u32(b, 0),
		MinY: u32(b, 4),
		MinZ: u32(b, 8),
		MaxX: u32(b, 12),
		MaxY: u32(b, 16),
		MaxZ: u32(b, 20),
	}, nil
}

// ---- 0x016B BoundingCylinderReport ----------------------------------------

type BoundingCylinderReport struct {
	X      uint32 `json:"x"`
	Y      uint32 `json:"y"`
	Z      uint32 `json:"z"`
	Radius uint32 `json:"radius"`
	Height uint32 `json:"height"`
}

func decodeBoundingCylinderReport(b []byte) (any, error) {
	if len(b) < 20 {
		return nil, errShort
	}
	return &BoundingCylinderReport{
		X:      u32(b, 0),
		Y:      u32(b, 4),
		Z:      u32(b, 8),
		Radius: u32(b, 12),
		Height: u32(b, 16),
	}, nil
}

// ---- 0x0178 QuaternionV3 --------------------------------------------------

type QuaternionV3 struct {
	SerialNumber   Serial `json:"serial_number"`
	NetworkTime    uint64 `json:"network_time"`
	X              int32  `json:"x"`
	Y              int32  `json:"y"`
	Z              int32  `json:"z"`
	W              int32  `json:"w"`
	QuaternionType uint8  `json:"quaternion_type"`
	Quality        uint16 `json:"quality"`
}

func decodeQuaternionV3(b []byte) (any, error) {
	if len(b) < 31 {
		return nil, errShort
	}
	return &QuaternionV3{
		SerialNumber:   Serial(u32(b, 0)),
		NetworkTime:    u64(b, 4),
		X:              i32(b, 12),
		Y:              i32(b, 16),
		Z:              i32(b, 20),
		W:              i32(b, 24),
		QuaternionType: u8(b, 28),
		Quality:        u16(b, 29),
	}, nil
}

// ---- 0x0179 UserDefinedV3 -------------------------------------------------

type UserDefinedV3 struct {
	SerialNumber Serial `json:"serial_number"`
	NetworkTime  uint64 `json:"network_time"`
	Payload      []byte `json:"payload"`
}

func decodeUserDefinedV3(b []byte) (any, error) {
	if len(b) < 12 {
		return nil, errShort
	}
	return &UserDefinedV3{
		SerialNumber: Serial(u32(b, 0)),
		NetworkTime:  u64(b, 4),
		Payload:      append([]byte(nil), b[12:]...),
	}, nil
}

// ---- 0x017A AccelerometerV3 -----------------------------------------------

type AccelerometerV3 struct {
	SerialNumber Serial `json:"serial_number"`
	NetworkTime  uint64 `json:"network_time"`
	X            int32  `json:"x"`
	Y            int32  `json:"y"`
	Z            int32  `json:"z"`
	Scale        uint8  `json:"scale"`
}

func decodeAccelerometerV3(b []byte) (any, error) {
	if len(b) < 25 {
		return nil, errShort
	}
	return &AccelerometerV3{
		SerialNumber: Serial(u32(b, 0)),
		NetworkTime:  u64(b, 4),
		X:            i32(b, 12),
		Y:            i32(b, 16),
		Z:            i32(b, 20),
		Scale:        u8(b, 24),
	}, nil
}

// ---- 0x017B GyroscopeV3 ---------------------------------------------------

type GyroscopeV3 struct {
	SerialNumber Serial `json:"serial_number"`
	NetworkTime  uint64 `json:"network_time"`
	X            int32  `json:"x"`
	Y            int32  `json:"y"`
	Z            int32  `json:"z"`
	Scale        uint16 `json:"scale"`
}

func decodeGyroscopeV3(b []byte) (any, error) {
	if len(b) < 26 {
		return nil, errShort
	}
	return &GyroscopeV3{
		SerialNumber: Serial(u32(b, 0)),
		NetworkTime:  u64(b, 4),
		X:            i32(b, 12),
		Y:            i32(b, 16),
		Z:            i32(b, 20),
		Scale:        u16(b, 24),
	}, nil
}

// ---- 0x017C MagnetometerV3 ------------------------------------------------

type MagnetometerV3 struct {
	SerialNumber Serial `json:"serial_number"`
	NetworkTime  uint64 `json:"network_time"`
	X            int32  `json:"x"`
	Y            int32  `json:"y"`
	Z            int32  `json:"z"`
	Scale        uint16 `json:"scale"`
}

func decodeMagnetometerV3(b []byte) (any, error) {
	if len(b) < 26 {
		return nil, errShort
	}
	return &MagnetometerV3{
		SerialNumber: Serial(u32(b, 0)),
		NetworkTime:  u64(b, 4),
		X:            i32(b, 12),
		Y:            i32(b, 16),
		Z:            i32(b, 20),
		Scale:        u16(b, 24),
	}, nil
}

// ---- 0x8009 ImageDiscoveryV1 ----------------------------------------------

type ImageDiscoveryV1 struct {
	Manufacturer     string  `json:"manufacturer"`
	Product          string  `json:"product"`
	RunningImageType uint8   `json:"running_image_type"`
	ImageInformation []Image `json:"image_information"`
}

func decodeImageDiscoveryV1(b []byte) (any, error) {
	if len(b) < 97 {
		return nil, errShort
	}
	out := &ImageDiscoveryV1{
		Manufacturer:     nullStrip(b[0:64]),
		Product:          nullStrip(b[64:96]),
		RunningImageType: u8(b, 96),
	}
	rest := b[97:]
	const itemSize = 53 // type(1)+version(32)+sha1(20)
	for len(rest) >= itemSize {
		img := Image{
			Type:    u8(rest, 0),
			Version: nullStrip(rest[1:33]),
			SHA1:    append([]byte(nil), rest[33:53]...),
		}
		out.ImageInformation = append(out.ImageInformation, img)
		rest = rest[itemSize:]
	}
	if len(rest) != 0 {
		return nil, fmt.Errorf("image_discovery_v1: trailing %d bytes", len(rest))
	}
	return out, nil
}

// ---- 0x802C TimedRxV5 -----------------------------------------------------

type TimedRxV5 struct {
	TxNT64             uint64            `json:"tx_nt64"`
	RxDT64             uint64            `json:"rx_dt64"`
	RxNT64             uint64            `json:"rx_nt64"`
	SourceSerialNumber Serial            `json:"source_serial_number"`
	SourceInterfaceID  uint8             `json:"source_interface_id"`
	SignalStrength     UWBSignalStrength `json:"signal_strength"`
	InterfaceID        uint8             `json:"interface_id"`
	TxNTQuality        uint8             `json:"tx_nt_quality"`
	RxNTQuality        uint8             `json:"rx_nt_quality"`
	RxPacketType       uint8             `json:"rx_packet_type"`
}

func decodeTimedRxV5(b []byte) (any, error) {
	if len(b) < 45 {
		return nil, errShort
	}
	return &TimedRxV5{
		TxNT64:             u64(b, 0),
		RxDT64:             u64(b, 8),
		RxNT64:             u64(b, 16),
		SourceSerialNumber: Serial(u32(b, 24)),
		SourceInterfaceID:  u8(b, 28),
		SignalStrength:     decodeSignalStrength(b, 29),
		InterfaceID:        u8(b, 41),
		TxNTQuality:        u8(b, 42),
		RxNTQuality:        u8(b, 43),
		RxPacketType:       u8(b, 44),
	}, nil
}

// ---- 0x802D TickV4 --------------------------------------------------------

type TickV4 struct {
	NT64        uint64 `json:"nt64"`
	DT64        uint64 `json:"dt64"`
	NTQuality   uint8  `json:"nt_quality"`
	InterfaceID uint8  `json:"interface_id"`
}

func decodeTickV4(b []byte) (any, error) {
	if len(b) < 18 {
		return nil, errShort
	}
	return &TickV4{
		NT64:        u64(b, 0),
		DT64:        u64(b, 8),
		NTQuality:   u8(b, 16),
		InterfaceID: u8(b, 17),
	}, nil
}

// ---- 0x802F PingV5 --------------------------------------------------------

type PingV5 struct {
	SourceSerialNumber Serial            `json:"source_serial_number"`
	Sequence           uint16            `json:"sequence"`
	BeaconType         uint8             `json:"beacon_type"`
	NTQuality          uint8             `json:"nt_quality"`
	DT64               uint64            `json:"dt64"`
	NT64               uint64            `json:"nt64"`
	SignalStrength     UWBSignalStrength `json:"signal_strength"`
	InterfaceID        uint8             `json:"interface_id"`
	Payload            []byte            `json:"payload"`
}

func decodePingV5(b []byte) (any, error) {
	if len(b) < 37 {
		return nil, errShort
	}
	return &PingV5{
		SourceSerialNumber: Serial(u32(b, 0)),
		Sequence:           u16(b, 4),
		BeaconType:         u8(b, 6),
		NTQuality:          u8(b, 7),
		DT64:               u64(b, 8),
		NT64:               u64(b, 16),
		SignalStrength:     decodeSignalStrength(b, 24),
		InterfaceID:        u8(b, 36),
		Payload:            append([]byte(nil), b[37:]...),
	}, nil
}

// ---- 0x80B2 TickV5 --------------------------------------------------------

type TickV5 struct {
	NT64           uint64 `json:"nt64"`
	DT64           uint64 `json:"dt64"`
	NTQuality      uint8  `json:"nt_quality"`
	InterfaceID    uint8  `json:"interface_id"`
	SequenceNumber uint8  `json:"sequence_number"`
}

func decodeTickV5(b []byte) (any, error) {
	if len(b) < 19 {
		return nil, errShort
	}
	return &TickV5{
		NT64:           u64(b, 0),
		DT64:           u64(b, 8),
		NTQuality:      u8(b, 16),
		InterfaceID:    u8(b, 17),
		SequenceNumber: u8(b, 18),
	}, nil
}

// ---- 0x80B3 TimedRxV6 -----------------------------------------------------

type TimedRxV6 struct {
	RxDT64             uint64            `json:"rx_dt64"`
	RxNT64             uint64            `json:"rx_nt64"`
	SourceSerialNumber Serial            `json:"source_serial_number"`
	SourceInterfaceID  uint8             `json:"source_interface_id"`
	SignalStrength     UWBSignalStrength `json:"signal_strength"`
	InterfaceID        uint8             `json:"interface_id"`
	RxNTQuality        uint8             `json:"rx_nt_quality"`
	RxPacketType       uint8             `json:"rx_packet_type"`
	TxSequence         uint8             `json:"tx_sequence"`
}

func decodeTimedRxV6(b []byte) (any, error) {
	if len(b) < 37 {
		return nil, errShort
	}
	return &TimedRxV6{
		RxDT64:             u64(b, 0),
		RxNT64:             u64(b, 8),
		SourceSerialNumber: Serial(u32(b, 16)),
		SourceInterfaceID:  u8(b, 20),
		SignalStrength:     decodeSignalStrength(b, 21),
		InterfaceID:        u8(b, 33),
		RxNTQuality:        u8(b, 34),
		RxPacketType:       u8(b, 35),
		TxSequence:         u8(b, 36),
	}, nil
}

// ---- 0x80C0 DeviceColor ---------------------------------------------------

type DeviceColor struct {
	SerialNumber Serial `json:"serial_number"`
	Red          uint8  `json:"red"`
	Green        uint8  `json:"green"`
	Blue         uint8  `json:"blue"`
	Alpha        uint8  `json:"alpha"`
}

func decodeDeviceColor(b []byte) (any, error) {
	if len(b) < 8 {
		return nil, errShort
	}
	return &DeviceColor{
		SerialNumber: Serial(u32(b, 0)),
		Red:          u8(b, 4),
		Green:        u8(b, 5),
		Blue:         u8(b, 6),
		Alpha:        u8(b, 7),
	}, nil
}

// ---- 0x80D4 DeviceStatusV3 ------------------------------------------------

type DeviceStatusV3 struct {
	SerialNumber           Serial         `json:"serial_number"`
	Memory                 uint32         `json:"memory"`
	Flags                  uint32         `json:"flags"`
	MinutesRemaining       uint16         `json:"minutes_remaining"`
	BatteryPercentage      uint8          `json:"battery_percentage"`
	Temperature            int8           `json:"temperature"`
	ProcessorUsage         uint8          `json:"processor_usage"`
	MissedPhaseCommands    uint16         `json:"missed_phase_commands"`
	MissedRecoveryCommands uint16         `json:"missed_recovery_commands"`
	MaxWideningFactor      uint16         `json:"max_widening_factor"`
	ErrorPatterns          []ErrorPattern `json:"error_patterns"`
}

func decodeDeviceStatusV3(b []byte) (any, error) {
	if len(b) < 23 {
		return nil, errShort
	}
	out := &DeviceStatusV3{
		SerialNumber:           Serial(u32(b, 0)),
		Memory:                 u32(b, 4),
		Flags:                  u32(b, 8),
		MinutesRemaining:       u16(b, 12),
		BatteryPercentage:      u8(b, 14),
		Temperature:            i8(b, 15),
		ProcessorUsage:         u8(b, 16),
		MissedPhaseCommands:    u16(b, 17),
		MissedRecoveryCommands: u16(b, 19),
		MaxWideningFactor:      u16(b, 21),
	}
	for _, p := range b[23:] {
		out.ErrorPatterns = append(out.ErrorPatterns, ErrorPattern{Pattern: p})
	}
	return out, nil
}

// ---- 0x80DA ClearDeviceColor ----------------------------------------------

type ClearDeviceColor struct {
	SerialNumber Serial `json:"serial_number"`
	Flags        uint8  `json:"flags"`
}

func decodeClearDeviceColor(b []byte) (any, error) {
	if len(b) < 5 {
		return nil, errShort
	}
	return &ClearDeviceColor{
		SerialNumber: Serial(u32(b, 0)),
		Flags:        u8(b, 4),
	}, nil
}

// ---- 0x80DB GeofencerZoneInfo ---------------------------------------------

type GeofencerZoneInfo struct {
	ZoneID     uint16         `json:"zone_id"`
	ZoneName   string         `json:"zone_name"`
	ZMin       int32          `json:"z_min"`
	ZMax       int32          `json:"z_max"`
	Hysteresis uint32         `json:"hysteresis"`
	Vertices   []XYCoordinate `json:"vertices"`
}

func decodeGeofencerZoneInfo(b []byte) (any, error) {
	if len(b) < 64 {
		return nil, errShort
	}
	out := &GeofencerZoneInfo{
		ZoneID:     u16(b, 0),
		ZoneName:   nullStrip(b[2:52]),
		ZMin:       i32(b, 52),
		ZMax:       i32(b, 56),
		Hysteresis: u32(b, 60),
	}
	rest := b[64:]
	const stride = 8
	for len(rest) >= stride {
		out.Vertices = append(out.Vertices, XYCoordinate{
			X: i32(rest, 0),
			Y: i32(rest, 4),
		})
		rest = rest[stride:]
	}
	if len(rest) != 0 {
		return nil, fmt.Errorf("geofencer_zone_info: trailing %d bytes", len(rest))
	}
	return out, nil
}

// ---- 0x80DC TagZoneInfo ---------------------------------------------------

type TagZoneInfo struct {
	SerialNumber Serial   `json:"serial_number"`
	ZoneList     []uint16 `json:"zone_list"`
}

func decodeTagZoneInfo(b []byte) (any, error) {
	if len(b) < 4 {
		return nil, errShort
	}
	rest := b[4:]
	if len(rest)%2 != 0 {
		return nil, fmt.Errorf("tag_zone_info: trailing %d bytes", len(rest)%2)
	}
	zones := make([]uint16, len(rest)/2)
	for i := range zones {
		zones[i] = u16(rest, i*2)
	}
	return &TagZoneInfo{
		SerialNumber: Serial(u32(b, 0)),
		ZoneList:     zones,
	}, nil
}

// ---- 0x80DD DrawPrism -----------------------------------------------------

type DrawPrism struct {
	Name     string         `json:"name"`
	Red      uint8          `json:"red"`
	Green    uint8          `json:"green"`
	Blue     uint8          `json:"blue"`
	Alpha    uint8          `json:"alpha"`
	ZMin     int32          `json:"z_min"`
	ZMax     int32          `json:"z_max"`
	Vertices []XYCoordinate `json:"vertices"`
}

func decodeDrawPrism(b []byte) (any, error) {
	if len(b) < 62 {
		return nil, errShort
	}
	out := &DrawPrism{
		Name:  nullStrip(b[0:50]),
		Red:   u8(b, 50),
		Green: u8(b, 51),
		Blue:  u8(b, 52),
		Alpha: u8(b, 53),
		ZMin:  i32(b, 54),
		ZMax:  i32(b, 58),
	}
	rest := b[62:]
	const stride = 8
	for len(rest) >= stride {
		out.Vertices = append(out.Vertices, XYCoordinate{
			X: i32(rest, 0),
			Y: i32(rest, 4),
		})
		rest = rest[stride:]
	}
	if len(rest) != 0 {
		return nil, fmt.Errorf("draw_prism: trailing %d bytes", len(rest))
	}
	return out, nil
}

// ---- 0x80DE ClearObject ---------------------------------------------------

type ClearObject struct {
	Name string `json:"name"`
}

func decodeClearObject(b []byte) (any, error) {
	if len(b) < 50 {
		return nil, errShort
	}
	return &ClearObject{Name: nullStrip(b[0:50])}, nil
}

// ---- helpers --------------------------------------------------------------

// nullStrip trims a byte slice at the first NUL byte and returns it as
// a string. Strings in CDP data items are NUL-terminated within a fixed
// or variable-length buffer.
func nullStrip(b []byte) string {
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}
