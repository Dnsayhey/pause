param(
  [Parameter(Mandatory = $true)]
  [string]$ShortcutPath,

  [Parameter(Mandatory = $true)]
  [string]$TargetPath,

  [Parameter(Mandatory = $true)]
  [string]$AppUserModelID
)

$ErrorActionPreference = 'Stop'

Add-Type -Language CSharp @"
using System;
using System.Runtime.InteropServices;

[ComImport]
[Guid("00021401-0000-0000-C000-000000000046")]
internal class CShellLink
{
}

[ComImport]
[InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
[Guid("000214F9-0000-0000-C000-000000000046")]
internal interface IShellLinkW
{
    void GetPath();
    void GetIDList();
    void SetIDList();
    void GetDescription();
    void SetDescription([MarshalAs(UnmanagedType.LPWStr)] string pszName);
    void GetWorkingDirectory();
    void SetWorkingDirectory([MarshalAs(UnmanagedType.LPWStr)] string pszDir);
    void GetArguments();
    void SetArguments([MarshalAs(UnmanagedType.LPWStr)] string pszArgs);
    void GetHotkey();
    void SetHotkey(short wHotkey);
    void GetShowCmd();
    void SetShowCmd(int iShowCmd);
    void GetIconLocation();
    void SetIconLocation([MarshalAs(UnmanagedType.LPWStr)] string pszIconPath, int iIcon);
    void SetRelativePath([MarshalAs(UnmanagedType.LPWStr)] string pszPathRel, int dwReserved);
    void Resolve(IntPtr hwnd, int fFlags);
    void SetPath([MarshalAs(UnmanagedType.LPWStr)] string pszFile);
}

[ComImport]
[InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
[Guid("0000010B-0000-0000-C000-000000000046")]
internal interface IPersistFile
{
    void GetClassID(out Guid pClassID);
    [PreserveSig]
    int IsDirty();
    void Load([MarshalAs(UnmanagedType.LPWStr)] string pszFileName, uint dwMode);
    void Save([MarshalAs(UnmanagedType.LPWStr)] string pszFileName, bool fRemember);
    void SaveCompleted([MarshalAs(UnmanagedType.LPWStr)] string pszFileName);
    void GetCurFile([MarshalAs(UnmanagedType.LPWStr)] out string ppszFileName);
}

[ComImport]
[InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
[Guid("886D8EEB-8CF2-4446-8D02-CDBA1DBDCF99")]
internal interface IPropertyStore
{
    uint GetCount();
    void GetAt(uint iProp, out PROPERTYKEY pkey);
    void GetValue(ref PROPERTYKEY key, out PROPVARIANT pv);
    void SetValue(ref PROPERTYKEY key, ref PROPVARIANT pv);
    void Commit();
}

[StructLayout(LayoutKind.Sequential)]
internal struct PROPERTYKEY
{
    public Guid fmtid;
    public uint pid;
}

[StructLayout(LayoutKind.Sequential)]
internal struct PROPVARIANT
{
    public ushort vt;
    public ushort wReserved1;
    public ushort wReserved2;
    public ushort wReserved3;
    public IntPtr pwszVal;

    public static PROPVARIANT FromString(string value)
    {
        return new PROPVARIANT
        {
            vt = 31,
            pwszVal = Marshal.StringToCoTaskMemUni(value),
        };
    }

    public void Clear()
    {
        if (pwszVal != IntPtr.Zero)
        {
            Marshal.FreeCoTaskMem(pwszVal);
            pwszVal = IntPtr.Zero;
        }
    }
}

public static class PauseShortcutInstaller
{
    public static void Create(string shortcutPath, string targetPath, string appUserModelID)
    {
        var shellLink = (IShellLinkW)new CShellLink();
        shellLink.SetPath(targetPath);
        shellLink.SetArguments(string.Empty);
        shellLink.SetWorkingDirectory(System.IO.Path.GetDirectoryName(targetPath));
        shellLink.SetIconLocation(targetPath, 0);

        var propertyStore = (IPropertyStore)shellLink;
        var appIdKey = new PROPERTYKEY
        {
            fmtid = new Guid("9F4C2855-9F79-4B39-A8D0-E1D42DE1D5F3"),
            pid = 5,
        };
        var value = PROPVARIANT.FromString(appUserModelID);
        try
        {
            propertyStore.SetValue(ref appIdKey, ref value);
            propertyStore.Commit();
        }
        finally
        {
            value.Clear();
        }

        ((IPersistFile)shellLink).Save(shortcutPath, true);
    }
}
"@

[PauseShortcutInstaller]::Create($ShortcutPath, $TargetPath, $AppUserModelID)
