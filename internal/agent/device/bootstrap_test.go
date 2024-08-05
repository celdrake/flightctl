package device

import (
	"context"
	"errors"
	"testing"

	"github.com/flightctl/flightctl/api/v1alpha1"
	"github.com/flightctl/flightctl/internal/agent/device/spec"
	"github.com/flightctl/flightctl/internal/agent/device/status"
	flightlog "github.com/flightctl/flightctl/pkg/log"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestBootstrapCheckRollback(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStatusManager := status.NewMockManager(ctrl)
	mockSpecManager := spec.NewMockManager(ctrl)

	b := &Bootstrap{
		statusManager: mockStatusManager,
		specManager:   mockSpecManager,
		log:           flightlog.NewPrefixLogger("test"),
	}

	ctx := context.TODO()
	bootedOS := "1.0.0"
	desiredOS := "2.0.0"

	t.Run("happy path", func(t *testing.T) {
		err := b.checkRollback(ctx, desiredOS, desiredOS)
		require.NoError(err)
	})

	t.Run("successfully handles no rollback", func(t *testing.T) {
		isRollingBack := false

		mockStatusManager.EXPECT().Update(ctx, gomock.Any()).Return(nil, nil)
		mockSpecManager.EXPECT().IsRollingBack(ctx).Return(isRollingBack, nil)

		err := b.checkRollback(ctx, bootedOS, desiredOS)
		require.NoError(err)
	})

	t.Run("successfully handles rollback", func(t *testing.T) {
		isRollingBack := true

		mockStatusManager.EXPECT().Update(ctx, gomock.Any()).Return(nil, nil)
		mockSpecManager.EXPECT().IsRollingBack(ctx).Return(isRollingBack, nil)
		mockSpecManager.EXPECT().Rollback().Return(nil)

		err := b.checkRollback(ctx, bootedOS, desiredOS)
		require.NoError(err)
	})

	t.Run("error checking rollback status", func(t *testing.T) {
		isRollingBack := false

		mockStatusManager.EXPECT().Update(ctx, gomock.Any()).Return(nil, nil)
		mockSpecManager.EXPECT().IsRollingBack(ctx).Return(isRollingBack, errors.New("rollback check failed"))

		err := b.checkRollback(ctx, bootedOS, desiredOS)
		require.Error(err)
	})

	t.Run("error during rollback", func(t *testing.T) {
		isRollingBack := true

		mockStatusManager.EXPECT().Update(ctx, gomock.Any()).Return(nil, nil)
		mockSpecManager.EXPECT().IsRollingBack(ctx).Return(isRollingBack, nil)
		mockSpecManager.EXPECT().Rollback().Return(errors.New("rollback failed"))

		err := b.checkRollback(ctx, bootedOS, desiredOS)
		require.Error(err)
	})

	t.Run("error updating status", func(t *testing.T) {
		isRollingBack := false

		mockStatusManager.EXPECT().Update(ctx, gomock.Any()).Return(nil, errors.New("update failed"))
		mockSpecManager.EXPECT().IsRollingBack(ctx).Return(isRollingBack, nil)

		err := b.checkRollback(ctx, bootedOS, desiredOS)
		require.NoError(err)
	})
}

func TestEnsureBootedOS(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStatusManager := status.NewMockManager(ctrl)
	mockSpecManager := spec.NewMockManager(ctrl)

	// Create a Bootstrap instance with the mocks
	b := &Bootstrap{
		statusManager: mockStatusManager,
		specManager:   mockSpecManager,
		log:           flightlog.NewPrefixLogger("test"),
	}

	ctx := context.TODO()
	desired := &v1alpha1.RenderedDeviceSpec{
		Os: &v1alpha1.DeviceOSSpec{
			Image: "desired-image",
		},
		RenderedVersion: "1",
	}

	t.Run("no desired OS image specified", func(t *testing.T) {
		emptyDesired := &v1alpha1.RenderedDeviceSpec{}
		err := b.ensureBootedOS(ctx, emptyDesired)
		require.NoError(err)
	})

	t.Run("no OS update in progress", func(t *testing.T) {
		isOSUpdate := false
		mockSpecManager.EXPECT().IsOSUpdate().Return(isOSUpdate, nil)

		err := b.ensureBootedOS(ctx, desired)
		require.NoError(err)
	})

	t.Run("OS image reconciliation failure", func(t *testing.T) {
		isOSUpdate := true
		isReconciled := false
		osReconciliationError := errors.New("reconciliation failed")

		mockSpecManager.EXPECT().IsOSUpdate().Return(isOSUpdate, nil)
		mockSpecManager.EXPECT().CheckOsReconciliation(ctx).Return("", isReconciled, osReconciliationError)

		err := b.ensureBootedOS(ctx, desired)
		require.Error(err)
	})

	t.Run("OS image not reconciled - triggers rollback", func(t *testing.T) {
		isOSUpdate := true
		isReconciled := false
		isRollingBack := true
		bootedImage := "unexpected-booted-image"

		mockSpecManager.EXPECT().IsOSUpdate().Return(isOSUpdate, nil)
		mockSpecManager.EXPECT().CheckOsReconciliation(ctx).Return(bootedImage, isReconciled, nil)
		mockSpecManager.EXPECT().IsRollingBack(ctx).Return(isRollingBack, nil)
		mockSpecManager.EXPECT().Rollback().Return(nil)
		mockStatusManager.EXPECT().Update(ctx, gomock.Any()).Return(nil, nil)

		err := b.ensureBootedOS(ctx, desired)
		require.NoError(err)
	})

	t.Run("OS image reconciled", func(t *testing.T) {
		isOSUpdate := true
		isReconciled := true
		bootedImage := "desired-image"

		mockSpecManager.EXPECT().IsOSUpdate().Return(isOSUpdate, nil)
		mockSpecManager.EXPECT().CheckOsReconciliation(ctx).Return(bootedImage, isReconciled, nil)
		mockSpecManager.EXPECT().Upgrade().Return(nil)
		mockStatusManager.EXPECT().Update(ctx, gomock.Any()).Return(nil, nil)

		err := b.ensureBootedOS(ctx, desired)
		require.NoError(err)
	})

	t.Run("error during upgrade", func(t *testing.T) {
		isOSUpdate := true
		isReconciled := true
		bootedImage := "desired-image"

		mockSpecManager.EXPECT().IsOSUpdate().Return(isOSUpdate, nil)
		mockSpecManager.EXPECT().CheckOsReconciliation(ctx).Return(bootedImage, isReconciled, nil)
		mockSpecManager.EXPECT().Upgrade().Return(errors.New("upgrade failed"))

		err := b.ensureBootedOS(ctx, desired)
		require.Error(err)
	})

	t.Run("error updating status", func(t *testing.T) {
		isOSUpdate := true
		isReconciled := true
		bootedImage := "desired-image"

		mockSpecManager.EXPECT().IsOSUpdate().Return(isOSUpdate, nil)
		mockSpecManager.EXPECT().CheckOsReconciliation(ctx).Return(bootedImage, isReconciled, nil)
		mockSpecManager.EXPECT().Upgrade().Return(nil)
		mockStatusManager.EXPECT().Update(ctx, gomock.Any()).Return(nil, errors.New("update status failed"))

		err := b.ensureBootedOS(ctx, desired)
		require.NoError(err)
	})
}
