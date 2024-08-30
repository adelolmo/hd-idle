# hd-idle

Reimplementation of _Christian Mueller's_ [hd-idle](http://hd-idle.sf.net) with some extra features.

`hd-idle` is a utility program for spinning-down external disks after a period of idle time. 
Since most external IDE disk enclosures don't support setting the IDE idle timer, 
a program like `hd-idle` is required to spin down idle disks automatically.

_Important note_: `hd-idle` is not compatible with the usage of disk monitoring tools like
**smartmontools**.

**Index**
* [Extra features](#extra-features)
  * [Support ATA commands](#support-ata-commands)
  * [Monitor the skew between monitoring cycles](#monitor-the-skew-between-monitoring-cycles)
  * [Resolve symlinks in runtime](#resolve-symlinks-in-runtime)
  * [Log disk spin up](#log-disk-spin-up)
  * [Use disk partitions or device mapper to calculate activity](#use-disk-partitions-or-device-mapper-to-calculate-activity)
* [Install](#Install)
  * [Precompiled binaries](#precompiled-binaries)
  * [Build from source](#build-from-source)
* [Run hd-idle](#run-hd-idle)
* [Configuration](#Configuration)
* [Understand the logs](#understand-the-logs)
  * [Standard log](#standard-log)
  * [Log file](#log-file)
* [Warning on spinning down disks](#warning-on-spinning-down-disks)
* [Troubleshot](#Troubleshot)
  * [Disks won't spin down](#disks-wont-spin-down)
  * [LUKS support](#luks-support)
  * [SCSI response not ok](#scsi-response-not-ok)

## Extra features

List of extra features compared to the original `hd-idle`:
 
### Support ATA commands

The implementation of `hd-idle` written by _Christian Mueller_ relies on the `SCSI` api to work.
When listing the drives by id, disks starting with `usb` stop using the original implementation, 
but disk starting with `ata` do not.

    $ ls /dev/disk/by-id/
    
    ata-WDC_WD40EZRX-
    ata-WDC_WD50EZRX-
    usb-WD_My_Book_1140_
    
[hdparm](https://en.wikipedia.org/wiki/Hdparm) on the other hand always stops the drives without any problems.
It uses `ATA` api calls to send disks to standby. `hd-idle` comes with `ATA` commands support to replicate `hdparm`'s api calls.

### Monitor the skew between monitoring cycles

Identify if the sleep took longer than expected and reset the spun down flag if it waited too long for the main loop sleep. 
This should capture suspend events as well as excessive machine load.

### Resolve symlinks in runtime

`hd-idle` can resolve disk symlinks also in runtime. Disks added after application's start won't be hidden. 

### Log disk spin up

Show in standard output when disks spin up. 

### Use disk partitions or device mapper to calculate activity

The disk activity is calculated by watching read/write changes on partition or device mapper level instead of disk level.
This is required for kernels newer than 5.4 LTS, because disk monitoring tools change read/write values on disk level,
although there's no real activity on the disk itself.
When using LUKS, activity will happen on the device mapper device mapped to the corresponding disk.

## Install

There are various ways of installing `hd-idle`:

### Precompiled binaries

Precompiled binaries for released versions are available in the 
[*releases*](https://github.com/adelolmo/hd-idle/releases) section.

### Build from source

To build `hd-idle` from the source code yourself you need to have a working
Go environment with [version 1.16 or greater installed](http://golang.org/doc/install).

Open a terminal and execute these commands:

    git clone https://github.com/adelolmo/hd-idle
    cd hd-idle
    make

On Debian you can also build the package yourself using `dpkg-buildpackage`:

    git clone https://github.com/adelolmo/hd-idle.git
    cd hd-idle
    dpkg-buildpackage -a armhf -us -uc -b
    
In the example above, the package is built for `armhf`, but you can build it also for the platforms `i386`, `amd64`, and `arm64` 
by substituting the parameter `-a`.
    
Then install the package:

    # dpkg -i ../hd-idle*.deb
    
## Run hd-idle

In order to run `hd-idle`, type: 

    $ hd-idle
    
This will start `hd-idle` with the default options, causing all `SCSI` 
(read: USB, Firewire, SCSI, ...) hard disks to spin down after 10 minutes of inactivity.

If the Debian package was installed, after editing `/etc/default/hd-idle` and enabling it (`START_HD_IDLE=true`), 
run hd-idle with:

    # systemctl start hd-idle
    
To enable `hd-idle` on reboot:

    # systemctl enable hd-idle    

Please note that `hd-idle` uses */proc/diskstats* to read disk statistics. If
this file is not present, `hd-idle` won't work.

In case of problems, use the debug option *-d* to get further information.

## Configuration

Command line options:

+ -a *name*              
                        Set device name of disks for subsequent idle-time
                        parameters *-i*. This parameter is optional in the
                        sense that there's a default entry for all disks
                        which are not named otherwise by using this
                        parameter. This can also be a symlink
                        (e.g. /dev/disk/by-uuid/...)
                         
+ -i *idle_time*          
                        Idle time in seconds for the currently named disk(s)
                        (-a *name*) or for all disks.
                        Setting this value to `0` will never spin down the disk(s).
                         
+ -c *command_type*       
                        Api call to stop the device. Possible values are `scsi`
                        (default value) and `ata`.

+ -p *power_condition*       
                        Power condition to send with the issued SCSI START STOP UNIT command. Possible values 
                        are `0-15` (inclusive). The default value of `0` works fine for disks accessible via the
                        SCSI layer (USB, IEEE1394, ...), but it will *NOT* work as intended with real SCSI / SAS disks.
                        A stopped SAS disk will not start up automatically on access, but requires a startup command for reactivation.
                        Useful values for  SAS disks are `2` for idle and `3` for standby. 

+ -s *symlink_policy*   
                        Set the policy to resolve symlinks for devices. If set 
                        to `0`, symlinks are resolved only on start. If set to `1`,
                        symlinks are also resolved on runtime until success.
                        By default symlinks are only resolved on start. If the 
                        symlink doesn't resolve to a device, the default
                        configuration will be applied.

+ -l *logfile*            
                        Name of logfile (written only after a disk has spun
                        up or down). Please note that this option might cause the
                        disk which holds the logfile to spin up just because
                        another disk had some activity. On single-disk systems,
                        this option should not cause any additional spinups.
                        On systems with more than one disk, the disk where the log
                        is written will be spun up. On raspberry based systems the 
                        log should be written to the SD card.

Miscellaneous options:

+ -t *disk*               
                        Spin-down the specified disk immediately and exit.
 
+ -d                      
                        Debug mode. It will print debugging info to
                        stdout/stderr (/var/log/syslog if started with systemctl)
                         
+ -h                      
                        Print usage information.

Regarding the parameter *-a*:

The parameter *-a* can be used to set a filter on the disk's device name (omit /dev/) 
for subsequent idle-time settings.

1) 
    A *-i* option before the first *-a* option will set the default idle time.

2) 
    In order to disable spin-down of disks per default, and then re-enable
    spin-down on selected disks, set the default idle time to 0.

    Example:
    ```
    hd-idle -i 0 -a sda -i 300 -a sdb -i 1200
    ```
    This example sets the default idle time to 0 (meaning hd-idle will never
    try to spin down a disk) and the default api command to `scsi`, then sets explicit 
    idle times for disks which have the string `sda` or `sdb` in their device name.
 
3) 
    The option *-c* allows to set the api call that sends the spindown command.
    Possible values are `scsi` (the default value) or `ata`.
    
    Example:
    ```
    hd-idle -i 0 -c ata -a sda -i 300 -a sdb -i 1200 -c scsi
    ```  
    This example sets the default idle time to 0 (meaning hd-idle will never
    try to spin down a disk) and the default api command to `ata`, then sets explicit 
    idle times for disks which have the string `sda` or `sdb` in their device name 
    and sets `sdb` to use `scsi` api command.

## Understand the logs

By default `hd-idle` only logs to the standard output. You can find them in the syslog if the application starts via service.

If you set the log file (`-l` flag) then the application writes extra details to it. (Check the [Configuration](#Configuration) section).

### Standard log

The standard log output registers two kinds of events:

* disk spin up
* disk spin down

```
Aug  8 00:14:55 enterprise hd-idle[9958]: sda spindown
Aug  8 00:14:55 enterprise hd-idle[9958]: sdb spindown
Aug  8 00:14:56 enterprise hd-idle[9958]: sdc spindown
Aug  8 00:17:55 enterprise hd-idle[9958]: sdb spinup
Aug  8 00:28:55 enterprise hd-idle[9958]: sdb spindown
```

### Log file

You can enable the log file with the flag `-l` followed by the log path. (Check the [Configuration](#Configuration) section).

This is the kind of entry shown in the log file:

```
date: 2020-07-30, time: 05:28:01, disk: sdc, running: 601, stopped: 76654
```

Explanation:
* `date` and `time` when the disk spins up.
* `disk` involved.
* `running` seconds the device was running before it spun down the last time.
* `stopped` seconds since last spin down. This is the time the disk was asleep before spinning up.

**Important Note:**

The log file is written after a full cycle of running-stopped-wakeup.

A bit more on `running` explained with the above example:

|     timestamp     |disk spin|    event    |new disk spin|         running          |                         stopped                         |
|:-----------------:|:-------:|:-----------:|:-----------:|:------------------------:|:-------------------------------------------------------:|
|2020-07-29 07:59:57|  down   |disk activity|     up      |            ?             |                            ?                            |
|2020-07-29 08:09:58|   up    | go to sleep |    down     |            -             |                            -                            |
|2020-07-30 05:28:01|  down   |disk activity|     up      |08:09:58 - 07:59:57 = 601s|2020-07-30 05:28:01 - 2020-07-29 08:09:58 = ~21h (76654s)|

Explanation:

At 07:59:57 the disk is on standby and hd-idle detects disk activity.

At 08:09:58 the disk is active and hd-idle determines inactivity of the disk and spins it down.

At 05:28:01 on the next day the disk is on standby and hd-idle detects disk activity. It writes on the log file 601s of previous disk spin up and ~21h of standby.


## Warning on spinning down disks

A word of caution: hard disks don't like spinning up too often. Laptop disks
are more robust in this respect than desktop disks but if you set your disks
to spin down after a few seconds you may damage the disk over time due to the
stress the spin-up causes on the spindle motor and bearings. It seems that
manufacturers recommend a minimum idle time of 3-5 minutes, the default in
`hd-idle` is 10 minutes.

You have been warned...

# Troubleshot

This section covers some usual issues that users face while using `hd-idle`.

## Disks won't spin down

Unfortunately, it's not possible to get `hd-idle` working alongside disk monitoring
tools like _smartmontools_. You have to disable those tools in order to 
get `hd-idle` working.

## LUKS support

Using encrypted disk or partitions with LUKS is supported by the use of symlinks.

1. Run the following command with you're disk mounted:
`sudo lsblk /dev/sd* -o PATH,FSSIZE,LABEL,UUID,PARTLABEL,PARTUUID,MODEL,SIZE,SERIAL,TYPE,WWN`

```
PATH                            FSSIZE LABEL UUID                                 PARTLABEL PARTUUID                             MODEL   SIZE SERIAL TYPE  WWN
/dev/sde                                                                                                                         ST400   3.7T ZGY0LB disk  0x5000c500a3d1d419
/dev/sde1                                    100e952e-0ffb-4b73-bb1a-8401d4fe56c0 dropbox   14a81aa8-c2c9-448e-967b-85d87dc9b488           1T        part  0x5000c500a3d1d419
/dev/sde1                                    100e952e-0ffb-4b73-bb1a-8401d4fe56c0 dropbox   14a81aa8-c2c9-448e-967b-85d87dc9b488           1T        part  0x5000c500a3d1d419
/dev/sde2                         2.6T three 175e2227-d24f-4ad0-9e42-2ddb8846682d           d2792423-3c07-44fe-ab6b-a1aca61c73a5         2.7T        part  0x5000c500a3d1d419
/dev/sde2                         2.6T three 175e2227-d24f-4ad0-9e42-2ddb8846682d           d2792423-3c07-44fe-ab6b-a1aca61c73a5         2.7T        part  0x5000c500a3d1d419
/dev/mapper/luks-100e952e-0ffb-4b73-bb1a-8401d4fe56c0
                               1007.8G dropbox
                                             649dd15e-6750-472c-8185-4d76bffc2490                                                       1024G        crypt 
```

You have to take symlinks that resolve to disk devices: `/dev/sd*`. 

In the example above `/dev/mapper/luks-100e952e-0ffb-4b73-bb1a-8401d4fe56c0` is the Path to the encrypted partition, 
which WWN is `0x5000c500a3d1d419`. 

2. Run the following command to see which devices the system has identified using `by-id`: 
`sudo ls -lv /dev/disk/by-id/`

Output:
```
lrwxrwxrwx 1 root root  9 Jul 18 15:56 ata-ST4000DM005-2DP166_ZGY0LBRB -> ../../sde
lrwxrwxrwx 1 root root 10 Jul 18 16:01 ata-ST4000DM005-2DP166_ZGY0LBRB-part1 -> ../../sde1
lrwxrwxrwx 1 root root 10 Jul 18 15:56 ata-ST4000DM005-2DP166_ZGY0LBRB-part2 -> ../../sde2
lrwxrwxrwx 1 root root  9 Jul 18 15:56 wwn-0x5000c500a3d1d419 -> ../../sde
lrwxrwxrwx 1 root root 10 Jul 18 16:01 wwn-0x5000c500a3d1d419-part1 -> ../../sde1
lrwxrwxrwx 1 root root 10 Jul 18 15:56 wwn-0x5000c500a3d1d419-part2 -> ../../sde2
```

Here we see that we can either use `ata-ST4000DM005-2DP166_ZGY0LBRB` or `wwn-0x5000c500a3d1d419` as symlinks.

3. Edit `/etc/default/hd-idle` to use the symlink you prefer. 
In my case, I went with the symlink using WWN (unique storage identifier), yet I could have chosen MODEL (device identifier) instead.

`HD_IDLE_OPTS='-i 0 -c ata -s 1 -l /var/log/hd-idle.log -a /dev/disk/by-id/wwn-0x5000c500a3d1d419 -i 600'`
Or
`HD_IDLE_OPTS='-i 0 -c ata -s 1 -l /var/log/hd-idle.log -a /dev/disk/by-id/ata-ST4000DM005-2DP166_ZGY0LBRB -i 600'`

## SCSI response not ok

You can find information about the issue here: [SCSI-response-not-ok](https://github.com/adelolmo/hd-idle/wiki/SCSI-response-not-ok)

## License

GNU General Public License v3.0, see [LICENSE](https://github.com/adelolmo/hd-idle/blob/master/LICENSE).
