/* Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2021
 *
 */

/*
 * arm64 utilities are in the webhooks package because they are used both
 * by validation and mutation webhooks.
 */
package webhooks

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"
)

var _false bool = false

const (
	defaultCPUModel = v1.CPUModeHostPassthrough
)

// ValidateVirtualMachineInstanceArm64Setting is validation function for validating-webhook
// 1. if setting bios boot
// 2. if use uefi secure boot
// 3. if use host-model for cpu model
// 4. if not use 'scsi', 'virtio' as disk bus
func ValidateVirtualMachineInstanceArm64Setting(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var statusCauses []metav1.StatusCause
	if spec.Domain.Firmware != nil && spec.Domain.Firmware.Bootloader != nil {
		if spec.Domain.Firmware.Bootloader.BIOS != nil {
			statusCauses = append(statusCauses, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueNotSupported,
				Message: "Arm64 does not support bios boot, please change to uefi boot",
				Field:   field.Child("domain", "firmware", "bootloader", "bios").String(),
			})
		}
		if spec.Domain.Firmware.Bootloader.EFI != nil {
			// When EFI is enable, secureboot is enabled by default, so here check two condition
			// 1 is EFI is enabled without Secureboot setting
			// 2 is both EFI and Secureboot enabled
			if spec.Domain.Firmware.Bootloader.EFI.SecureBoot == nil || (spec.Domain.Firmware.Bootloader.EFI.SecureBoot != nil && *spec.Domain.Firmware.Bootloader.EFI.SecureBoot) {
				statusCauses = append(statusCauses, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueNotSupported,
					Message: "UEFI secure boot is currently not supported on aarch64 Arch",
					Field:   field.Child("domain", "firmware", "bootloader", "efi", "secureboot").String(),
				})
			}
		}
	}
	if spec.Domain.CPU != nil && (&spec.Domain.CPU.Model != nil) && spec.Domain.CPU.Model == "host-model" {
		statusCauses = append(statusCauses, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueNotSupported,
			Message: "Arm64 not support host model well",
			Field:   field.Child("domain", "cpu", "model").String(),
		})
	}
	if spec.Domain.Devices.Disks != nil {
		// checkIfBusAvailable: if bus type is nil, virtio, scsi return true, otherwise, return false
		checkIfBusAvailable := func(bus v1.DiskBus) bool {
			if bus == "" || bus == v1.DiskBusVirtio || bus == v1.DiskBusSCSI {
				return true
			}
			return false
		}

		for i, disk := range spec.Domain.Devices.Disks {
			if disk.Disk != nil && !checkIfBusAvailable(disk.Disk.Bus) {
				statusCauses = append(statusCauses, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueNotSupported,
					Message: "Arm64 not support this disk bus type, please use virtio or scsi",
					Field:   field.Child("domain", "devices", "disks").Index(i).Child("disk", "bus").String(),
				})
			}
			if disk.CDRom != nil && !checkIfBusAvailable(disk.CDRom.Bus) {
				statusCauses = append(statusCauses, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueNotSupported,
					Message: "Arm64 not support this disk bus type, please use virtio or scsi",
					Field:   field.Child("domain", "devices", "disks").Index(i).Child("cdrom", "bus").String(),
				})
			}
			if disk.LUN != nil && !checkIfBusAvailable(disk.LUN.Bus) {
				statusCauses = append(statusCauses, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueNotSupported,
					Message: "Arm64 not support this disk bus type, please use virtio or scsi",
					Field:   field.Child("domain", "devices", "disks").Index(i).Child("lun", "bus").String(),
				})
			}
		}
	}
	return statusCauses
}

// setDefaultCPUModel set default cpu model to host-passthrough
func setDefaultCPUModel(vmi *v1.VirtualMachineInstance) {
	if vmi.Spec.Domain.CPU == nil {
		vmi.Spec.Domain.CPU = &v1.CPU{}
	}

	if vmi.Spec.Domain.CPU.Model == "" {
		vmi.Spec.Domain.CPU.Model = defaultCPUModel
	}
}

// setDefaultBootloader set default bootloader to uefi boot
func setDefaultBootloader(vmi *v1.VirtualMachineInstance) {
	if vmi.Spec.Domain.Firmware == nil || vmi.Spec.Domain.Firmware.Bootloader == nil {
		if vmi.Spec.Domain.Firmware == nil {
			vmi.Spec.Domain.Firmware = &v1.Firmware{}
		}
		if vmi.Spec.Domain.Firmware.Bootloader == nil {
			vmi.Spec.Domain.Firmware.Bootloader = &v1.Bootloader{}
		}
		vmi.Spec.Domain.Firmware.Bootloader.EFI = &v1.EFI{}
		vmi.Spec.Domain.Firmware.Bootloader.EFI.SecureBoot = &_false
	}
}

// setDefaultDisksBus set default Disks Bus, because sata is not supported by qemu-kvm of Arm64
func setDefaultDisksBus(vmi *v1.VirtualMachineInstance) {
	bus := v1.DiskBusVirtio

	for i := range vmi.Spec.Domain.Devices.Disks {
		disk := &vmi.Spec.Domain.Devices.Disks[i].DiskDevice

		if disk.Disk != nil && disk.Disk.Bus == "" {
			disk.Disk.Bus = bus
		}
		if disk.CDRom != nil && disk.CDRom.Bus == "" {
			disk.CDRom.Bus = bus
		}
		if disk.LUN != nil && disk.LUN.Bus == "" {
			disk.LUN.Bus = bus
		}
	}

}

// SetVirtualMachineInstanceArm64Defaults is mutating function for mutating-webhook
func SetVirtualMachineInstanceArm64Defaults(vmi *v1.VirtualMachineInstance) {
	setDefaultCPUModel(vmi)
	setDefaultBootloader(vmi)
	setDefaultDisksBus(vmi)
}
