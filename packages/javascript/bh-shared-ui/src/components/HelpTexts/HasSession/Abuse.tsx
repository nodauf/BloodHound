// Copyright 2023 Specter Ops, Inc.
// 
// Licensed under the Apache License, Version 2.0
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// 
//     http://www.apache.org/licenses/LICENSE-2.0
// 
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// 
// SPDX-License-Identifier: Apache-2.0

import { FC } from 'react';
import { Typography } from '@mui/material';

const Abuse: FC = () => {
    return (
        <>
            <Typography variant='body1'>Password Theft</Typography>
            <Typography variant='body2'>
                When a user has a session on the computer, you may be able to obtain credentials for the user via
                credential dumping or token impersonation. You must be able to move laterally to the computer, have
                administrative access on the computer, and the user must have a non-network logon session on the
                computer.
            </Typography>
            <Typography variant='body2'>
                Once you have established a Cobalt Strike Beacon, Empire agent, or other implant on the target, you can
                use mimikatz to dump credentials of the user that has a session on the computer. While running in a high
                integrity process with SeDebugPrivilege, execute one or more of mimikatz's credential gathering
                techniques (e.g.: sekurlsa::wdigest, sekurlsa::logonpasswords, etc.), then parse or investigate the
                output to find clear-text credentials for other users logged onto the system.
            </Typography>
            <Typography variant='body2'>
                You may also gather credentials when a user types them or copies them to their clipboard! Several
                keylogging capabilities exist, several agents and toolsets have them built-in. For instance, you may use
                meterpreter's "keyscan_start" command to start keylogging a user, then "keyscan_dump" to return the
                captured keystrokes. Or, you may use PowerSploit's Invoke-ClipboardMonitor to periodically gather the
                contents of the user's clipboard.
            </Typography>

            <Typography variant='body1'>Token Impersonation</Typography>
            <Typography variant='body2'>
                You may run into a situation where a user is logged onto the system, but you can't gather that user's
                credential. This may be caused by a host-based security product, lsass protection, etc. In those
                circumstances, you may abuse Windows' token model in several ways. First, you may inject your agent into
                that user's process, which will give you a process token as that user, which you can then use to
                authenticate to other systems on the network. Or, you may steal a process token from a remote process
                and start a thread in your agent's process with that user's token. For more information about token
                abuses, see the References tab.
            </Typography>
            <Typography variant='body2'>
                User sessions can be short lived and only represent the sessions that were present at the time of
                collection. A user may have ended their session by the time you move to the computer to target them.
                However, users tend to use the same machines, such as the workstations or servers they are assigned to
                use for their job duties, so it can be valuable to check multiple times if a user session has started.
            </Typography>
        </>
    );
};

export default Abuse;
