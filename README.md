# Guild Wars 2 Addon Debugger

Tool for debugging the myriad of possible issues that could go wrong with setting up addons for Guild Wars 2.
Collects a variety of data that is typically asked for when a user asks for support, and generates a report that can be shared for assistance.

## Usage

You might have been asked to run this tool to help you fix a problem.

1. [Download the latest version of the tool](https://github.com/cheahjs/gw2-addon-setup-debug/releases/latest).
1. Run the tool. You will be asked if you want to allow the tool to run as admin - this is recommended to prevent issues with gathering data.
1. Follow the step by step instructions provided.
1. At the end, save the report and share it to allow people help fix your issue.

## Features

Collects the following information:

1. The path the user thinks GW2 is installed in
1. A directory listing of the GW2 directory
1. A list of DLLs with the following information:
   1. The md5 checksum
   1. The product name and description
   1. The file version and description
   1. What DLL the tool thinks it is, supporting:
       1. [arcdps](https://www.deltaconnected.com/arcdps/) and its addons
       1. [addon-loader](https://github.com/gw2-addon-loader/loader-core) and its addons
       1. [nexus](https://raidcore.gg/Nexus) and its addons
       1. GW2Load and its addons
       1. [Reshade](https://reshade.me/)
       1. Generic D3D11/DXGI shims
   1. If the DLL has the Internet zone identifier that shows up as a blocked file
1. Information about the currently running GW2 process - the user is asked to launch this themselves to identify multiple installations
   1. The executable path
   1. The working directory
   1. The command-line arguments
   1. The currently loaded modules

## Screenshots

![gw2-addon-setup-debug_AcWmkZGmQL](https://github.com/user-attachments/assets/5dde945d-71a1-4d4a-9173-f7d2b16fd7c2)
