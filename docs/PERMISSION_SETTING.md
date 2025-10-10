# Permission setting for rootless usage of Remoteproc Runtime

For non-root users to use Remoteproc Runtime, the remoteproc driver and the container engine must be accessible for this user.

## How to configure rootless access to the remoteproc driver sysfs

Usually, the remoteproc sysfs entries are only accessible by root. To change this setting, follow the below instructions:

1. Create a group and add your user:

   ```
   sudo groupadd remoteproc
   sudo usermod -aG remoteproc "$USER"
   ```

   Log out and log back in to refresh group membership

2. Use systemd-tmpfiles to set mode/owner on every boot (and re-apply easily):

As a virtual filesystem, by default permission settings in the sysfs are not persisted across reboot.

   Create `/etc/tmpfiles.d/remoteproc.conf` with following:

   ```
   f /sys/class/remoteproc/remoteproc0/state                0664  root remoteproc -   -
   f /sys/class/remoteproc/remoteproc0/firmware             0664  root remoteproc -   -
   f /sys/class/remoteproc/remoteproc0/name                 0664  root remoteproc -   -
   ```

   Add similar lines for each additional remoteproc device (e.g., remoteproc1, remoteproc2, etc.) as needed.

3. Apply the change in remoteproc.conf:
   ```
   sudo systemd-tmpfiles --create /etc/tmpfiles.d/remoteproc.conf
   ```
4. Log in as a user in the remoteproc group and try the following commands to make sure that you can access the remoteproc driver as this user:
   ```
   # read state
   cat /sys/class/remoteproc/remoteproc0/state
   # start/stop
   echo start | tee /sys/class/remoteproc/remoteproc0/state
   echo stop  | tee /sys/class/remoteproc/remoteproc0/state
   ```

## Ensure your container engine is accessible by the user

The user must be able to access the container engine. For example, if you are using Docker, you need to add the user to the `docker` group:

```
sudo usermod -aG docker "$USER"
```

After running the command above, log out and log in again to refresh group membership.
