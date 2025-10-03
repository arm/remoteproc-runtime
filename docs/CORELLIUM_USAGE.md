# Setting up Corellium Virtual device

1. You need to make an account at [Corellium](https://app.avh.corellium.com/login). If you are new to Corellium, you can sign up with Arm Account to activate a free trial to try out remoteproc runtime.
2. Create new device
   ![setup corellium step 1](./images/setup_corellium1.png)
3. Select Default Project
   ![setup corellium step 2](./images/setup_corellium2.png)
4. Select i.MX93 board
   ![setup corellium step 3](./images/setup_corellium3.png)
5. Use [an image with necessary modules (such as docker-compose) installed](./YOCTO_BUILD_INST.md). Refer to [the Corellium documentation](https://www.corellium.com/blog/building-firmware-package-corellium-atlas) to make a compatible kernel.
   ℹ️ We recommend using the device tree provided by Corellium and not replacing it.
6. Upload your own kernel package to the custom image section
   ![setup corellium step 4](./images/setup_corellium4.png)
7. The Virtual i.MX93 will now be ready for testing. Return to [USAGE.md's Containered Shim section](./USAGE.md#containerd-shim) or [Container Runtime section](./USAGE.md#container-runtime-️-wip)
