package cdp

import "fmt"

// init builds the type registry. One entry per supported data item type ID;
// each entry knows the Go name, the NATS subject token (lowercase, version
// suffix stripped), and the decoder.
func init() {
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
	registry[0x015A] = registryEntry{"NtRealTimeMappingV1", "nt_realtime_mapping", decodeNtRealTimeMappingV1}
	registry[0x0160] = registryEntry{"BootloadProgress", "bootload_progress", decodeBootloadProgress}
	registry[0x0164] = registryEntry{"PolarCoordinatesV1", "polar_coordinates", decodePolarCoordinatesV1}
	registry[0x0171] = registryEntry{"ImageDiscoveryV2", "image_discovery", decodeImageDiscoveryV2}

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
