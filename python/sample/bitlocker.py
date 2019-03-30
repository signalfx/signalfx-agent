import wmi
import pythoncom

def run(config, output):
    # because run is called in a thread it is necessary to explicitly initialize
    # the COM libraries
    pythoncom.CoInitializeEx(pythoncom.COINIT_APARTMENTTHREADED)

    # if the bitlocker drive encryption feature is not installed then the attempt
    # to connect to the COM Object will raise an exception
    try:
        # BDE is installed
        mve = wmi.WMI(moniker="//./root/cimv2/security/microsoftvolumeencryption")
        ev_list = mve.Win32_EncryptableVolume()
        for ev in ev_list:
            drive = ev.DriveLetter
            bde_enabled = 0
            bde_locked = 0
            if ev.IsVolumeInitializedForProtection:
                bde_enabled = 1
                bde_locked = 1 if (ev.ProtectionStatus == 2) else 0
            output.send_gauge("bitlocker_drive_encryption.enabled", bde_enabled, {"volume": drive})
            output.send_gauge("bitlocker_drive_encryption.locked", bde_locked, {"volume": drive})
    except Exception:
        # BDE is not installed, report enabled = 0, locked = 0 for all drives
        mwmi = wmi.WMI()
        ld_list = mwmi.Win32_LogicalDisk()
        for ld in ld_list:
            drive = ld.DeviceID
            output.send_gauge("bitlocker_drive_encryption.enabled", 0, {"volume": drive})
            output.send_gauge("bitlocker_drive_encryption.locked", 0, {"volume": drive})
