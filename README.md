# hd-idle

This is a port to Go of Christian Mueller's [hd-idle](http://hd-idle.sf.net)

`hd-idle` is a utility program for spinning-down external disks after a period
of idle time. Since most external IDE disk enclosures don't support setting
the IDE idle timer, a program like `hd-idle` is required to spin down idle
disks automatically.

A word of caution: hard disks don't like spinning up too often. Laptop disks
are more robust in this respect than desktop disks but if you set your disks
to spin down after a few seconds you may damage the disk over time due to the
stress the spin-up causes on the spindle motor and bearings. It seems that
manufacturers recommend a minimum idle time of 3-5 minutes, the default in
`hd-idle` is 10 minutes.

One more word of caution: `hd-idle` will spin down any disk accessible via the
`SCSI` layer (USB, IEEE1394, ...) but it will NOT work with real `SCSI` disks
because they don't spin up automatically. Thus it's not called scsi-idle and
I don't recommend using it on a real `SCSI` system unless you have a kernel
patch that automatically starts the `SCSI` disks after receiving a sense buffer
indicating the disk has been stopped. Without such a patch, real `SCSI` disks
won't start again and you can as well pull the plug.

You have been warned...

## Background

The motivation to port `hd-idle` to Go comes directly from my lack of knowledge in C
and the need to use `ATA` api to set devices to stop.

The original `hd-idle` written by Christian Mueller relies on the `SCSI` api to work.
When listing the drives by id, disk starting with `usb` will stop using the original 
implementation, but disk starting with `ATA` will not.

    $ ls /dev/disk/by-id/
    
    ata-WDC_WD40EZRX-
    ata-WDC_WD50EZRX-
    usb-WD_My_Book_1140_
    
[hdparm](https://en.wikipedia.org/wiki/Hdparm) on the other hand was able to stop always the three drives without any problems.
It uses `ATA` api calls to do the job. So my idea was to replicate `hdparm`'s api call 
and add it to `hd-idle` itself.

## Install

There are various ways of installing `hd-idle`

### Precompiled binaries

Precompiled binaries for released versions are available in the 
[*releases*](https://github.com/adelolmo/hd-idle/releases) section.

### Building from source

To build `hd-idle` from the source code yourself you need to have a working
Go environment with [version 1.8 or greater installed](http://golang.org/doc/install).

You can directly use the `go` tool to download and install the `hd-idle` 
binaries into your `GOPATH`:

    $ go get github.com/adelolmo/hd-idle
    $ hd-idle

On Debian you can also clone the repository yourself and build using `dpkg-buildpackage`.

    $ mkdir -p $GOPATH/src/github.com/adelolmo
    $ cd $GOPATH/src/github.com/adelolmo
    $ git clone https://github.com/adelolmo/hd-idle.git
    $ cd hd-idle
    $ dpkg-buildpackage -us -uc -b
    
Then install the package:

    # dpkg -i ../hd-idle*.deb
    
## Running hd-idle

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
                         
+ -c *command_type*       
                        Api call to stop the device. Possible values are `scsi`
                        (default value) and `ata`.

+ -s *symlink_policy*   
                        Set the policy to resolve symlinks for devices. If set 
                        to `0`, symlinks are resolve only on start. If set to `1`,
                        symlinks are also resolved on runtime until success.
                        By default symlinks are only resolve on start. If the 
                        symlink doesn't resolve to a device, the default
                        configuration will be applied.

+ -l *logfile*            
                        Name of logfile (written only after a disk has spun
                        up or spun down). Please note that this option might cause the
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
    try to spin down a disk) and default `scsi` api command, then sets explicit 
    idle times for disks which have the string `sda` or `sdb` in their device name.
 
3) 
    The option *-c* allows to set the api call that sends the spindown command.
    Possible values are `scsi` (the default value) or `ata`.
    
    Example:
    ```
    hd-idle -i 0 -c ata -a sda -i 300 -a sdb -i 1200 -c scsi
    ```  
    This example sets the default idle time to 0 (meaning hd-idle will never
    try to spin down a disk) and default `ata` api command, then sets explicit 
    idle times for disks which have the string `sda` or `sdb` in their device name 
    and sets `sdb` to use `scsi` api command.

## License

GNU General Public License v3.0, see [LICENSE](https://github.com/adelolmo/hd-idle/blob/master/LICENSE).
