# i.MX93 board caveats
The Remoteproc driver for i.MX93 has some limitations. Workarounds may be needed in case of unexpected issues. Below are troubleshooting guidelines for common problems.

## My board stops responding when I start the M-class core using Remoteproc

### Example: How to reproduce
1. Copy `your-firmware.elf` to `/lib/firmware` on the board.
2. Use the Remoteproc framework to load the firmware:
```
echo your-firmware.elf > /sys/class/remoteproc/remoteproc0/firmware
echo start > /sys/class/remoteproc/remoteproc0/state
```
3. The board becomes unresponsive (neither UART responds).

### How to fix
Various sources ([NXP docs](https://www.nxp.com/document/guide/getting-started-with-frdm-imx93:GS-FRDM-IMX93?section=cortex-m33-enablement), [NXP forum](https://community.nxp.com/t5/i-MX-Processors/IMX8MP-EVK-bone-stock-freezes-upon-loading-Cortex-M7-RPMSG/m-p/1627688#M203714), [fs-net forum](https://forum.fs-net.de/index.php?thread/4817-starting-m7-from-linux-freezes-the-system-u-boot-prepare-mcore-is-missing/%5D)) indicate that you must boot the M-class core before starting the program. This is done from the U-Boot menu (press any key over UART during boot):
```
u-boot=> run prepare_mcore
u-boot=> boot
```
After performing this step, retry the above steps or your Docker run. The board should not lock up, and the firmware should load successfully.

> **Note:** This pre-boot step is not required for example programs built with MCUXpresso SDK, such as `baremetal-hello-world.elf` or `freertos-counter.elf`.

## NULL Pointer Dereference Error from Remoteproc Driver
A common cause of this issue is that your firmware does not include a resource table. The following example demonstrates the difference between firmware with and without a resource table, and how it affects Remoteproc operation.

### Example: How to reproduce
1. Copy both `your-firmware-with-resource-table.elf` and `your-firmware-without-resource-table.elf` to `/lib/firmware` on the board.
2. Load each firmware using the Remoteproc framework:
```
echo your-firmware-with-resource-table.elf > /sys/class/remoteproc/remoteproc0/firmware
echo start > /sys/class/remoteproc/remoteproc0/state
# ... wait ...
echo stop > /sys/class/remoteproc/remoteproc0/state
# ... wait ...
echo your-firmware-without-resource-table.elf > /sys/class/remoteproc/remoteproc0/firmware
echo start > /sys/class/remoteproc/remoteproc0/state
```
3. Observe the following error on the Linux host:
```
NULL pointer dereference at virtual address 0000000000000000
```
Remoteproc becomes unresponsive:
```
can't lock rproc imx-rproc: -4
```

### How to fix
Ensure your firmware always contains a resource table.

You can verify this by running:
```
readelf -S your-firmware.elf
```
Check for a `.resource_table` section in your ELF file. If it is missing, your firmware may not work with Remoteproc on the i.MX93 board.

> **Tip:** Most RTOSes and SDKs (e.g., MCUXpresso, FreeRTOS) generate a resource table automatically when RPMsg or Remoteproc support is enabled. For bare-metal or custom firmware, you must manually define and place a resource table in your code. Refer to NXP documentation for details.