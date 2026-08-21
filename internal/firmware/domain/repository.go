package domain

import "context"

type Repository interface {
	SaveFirmware(ctx context.Context, firmware Firmware) error
	GetFirmware(ctx context.Context, id string) (Firmware, error)
	ListFirmware(ctx context.Context, productID string, offset, limit int) ([]Firmware, int, error)

	SaveTask(ctx context.Context, task UpgradeTask) error
	GetTask(ctx context.Context, id string) (UpgradeTask, error)
	ListTasks(ctx context.Context, filter Filter, offset, limit int) ([]UpgradeTask, int, error)

	SaveReceipt(ctx context.Context, receipt DeviceReceipt) error
	GetReceipt(ctx context.Context, id string) (DeviceReceipt, error)
	ReceiptsForDevice(ctx context.Context, deviceID string, limit int) ([]DeviceReceipt, error)
	ActiveReceiptForDevice(ctx context.Context, deviceID string) (DeviceReceipt, error)
}
