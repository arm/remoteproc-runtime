# Permission setting for non-root users' usage of Remoteproc Runtime

For non-root users to use Remoteproc Runtime, the remoteproc driver and the container engine must be accessible for this user.

## How to set Remoteproc Runtime to be accessible by non-root users

### 1. Make remoteproc driver accessible to the user

Usually, remoteproc driver can only be accessible to root. To change this setting, follow the below instructions:

1. Create a group and add your user:

   ```sh
   sudo groupadd remoteproc
   sudo usermod -aG remoteproc "$USER"
   ```

   Log out and log back in to refresh group membership

2. Use systemd-tmpfiles to set mode/owner on every boot (and re-apply easily):

   Create /etc/tmpfiles.d/remoteproc.conf with following:

   ```
   f /sys/class/remoteproc/remoteproc0/state                0664  root remoteproc -   -
   f /sys/class/remoteproc/remoteproc0/firmware             0664  root remoteproc -   -
   f /sys/class/remoteproc/remoteproc0/name                 0664  root remoteproc -   -
   ```

   Add similar lines for each additional remoteproc device (e.g., remoteproc1, remoteproc2, etc.) as needed.

3. Apply the change in remoteproc.conf using root permission:
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

### 2. Make sure the user's instance is alive

1. Check what your UID is by running in your user session terminal:
   ```sh
   id -u
   ```
2. Ensure the instance of the user is alive in your user session terminal.
   ```sh
   systemctl --user status
   ```
   If you get:
   ```
   Failed to connect to bus: No medium found
   ```
   it means your D-Bus socket is not set right. Try the commands below, and try `systemctl --user status` again.
   ```sh
   export XDG_RUNTIME_DIR=/run/user/<uid>
   export DBUS_SESSION_BUS_ADDRESS=unix:path=$XDG_RUNTIME_DIR/bus
   ```

### 3. Set the firmware path to somewhere accessible by the user

1. Ensure that the path of the folder that contains your firmware is written to `/sys/module/firmware_class/parameters/path`. You need root permission for this.
   ```sh
   sudo echo <your firmware folder path> > /sys/module/firmware_class/parameters/path
   ```
