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
import { Link, Box } from '@mui/material';

const References: FC = () => {
    return (
        <Box sx={{ overflowX: 'auto' }}>
            <Link target='_blank' rel='noopener' href='https://github.com/BloodHoundAD/BARK'>
                Andy Robbins - BARK.ps1
            </Link>
            <br />
            <Link
                target='_blank'
                rel='noopener'
                href='https://posts.specterops.io/managed-identity-attack-paths-part-1-automation-accounts-82667d17187a'>
                Andy Robbins - Managed Identity Attack Paths, Part 1: Automation Accounts
            </Link>
        </Box>
    );
};

export default References;
