# Permission setting for non-root users' usage of Remoteproc Runtime

For non-root users to use Remoteproc Runtime, the remoteproc driver and the container engine must be accessible for this user.

## How to set Remoteproc Runtime to be accessible by non-root users

### 1. Make remoteproc driver accessible to the user

By default, the remoteproc device can only be accessible by root.

1. Create a group and add the user:

   ```sh
   sudo groupadd remoteproc
   sudo usermod -aG remoteproc "$USER"
   ```

   Log out and log back in to refresh group membership

2. Use systemd-tmpfiles to set mode/owner on every boot:

   Create /etc/tmpfiles.d/remoteproc.conf and add relevant remoteproc driver filesystem.
   Check the remoteproc name and make sure the correct remote processor is revealed to the group:

   ```
   f /sys/class/remoteproc/remoteproc0/state                0664  root remoteproc -   -
   f /sys/class/remoteproc/remoteproc0/firmware             0664  root remoteproc -   -
   f /sys/class/remoteproc/remoteproc0/name                 0664  root remoteproc -   -
   ```

   Add similar lines for each additional remoteproc device (e.g., remoteproc1, remoteproc2, etc.) as needed. On each new boot, the remoteproc processor number may be different depending on the driver probe order. It is recommended that this file is checked on each boot to reveal correct processor to correct boot before applying the configuration.

3. Apply the change in remoteproc.conf. This needs to be done on each boot:
   ```sh
   sudo systemd-tmpfiles --create /etc/tmpfiles.d/remoteproc.conf
   ```
4. Log in as a user in the remoteproc group and try the following commands to make sure that you can access the remoteproc driver as this user:
   ```sh
   # read state
   cat /sys/class/remoteproc/remoteproc0/state
   # start/stop
   echo start | tee /sys/class/remoteproc/remoteproc0/state
   echo stop  | tee /sys/class/remoteproc/remoteproc0/state
   ```

### 2. Set the firmware path to somewhere accessible by the user

1. Ensure that the path of the folder that contains your firmware is written to `/sys/module/firmware_class/parameters/path`. You need root permission for this.
   ```sh
   echo <your firmware folder path> | sudo tee /sys/module/firmware_class/parameters/path
   ```
