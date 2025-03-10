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

import { faCropAlt } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Box } from '@mui/material';
import { useSigma } from '@react-sigma/core';
import { random } from 'graphology-layout';
import forceAtlas2 from 'graphology-layout-forceatlas2';
import isEmpty from 'lodash/isEmpty';
import { FC } from 'react';
import GraphButton from 'src/components/GraphButton';
import { GraphButtonProps } from 'src/components/GraphButton/GraphButton';
import { resetCamera } from 'src/ducks/graph/utils';
import { RankDirection, layoutDagre } from 'src/hooks/useLayoutDagre/useLayoutDagre';

interface GraphButtonsProps {
    rankDirection?: RankDirection;
    options?: GraphButtonOptions;
    nonLayoutButtons?: GraphButtonProps[];
}

export type GraphButtonOptions = {
    standard: boolean;
    sequential: boolean;
};

const GraphButtons: FC<GraphButtonsProps> = ({ rankDirection, options, nonLayoutButtons }) => {
    if (isEmpty(options)) options = { standard: false, sequential: false };
    const { standard, sequential } = options;

    const sigma = useSigma();
    const graph = sigma.getGraph();
    const { assign: assignDagre } = layoutDagre(
        {
            graph: {
                rankdir: rankDirection || RankDirection.LEFT_RIGHT,
                ranksep: rankDirection === RankDirection.LEFT_RIGHT ? 500 : 50,
            },
        },
        graph
    );

    const runSequentialLayout = (): void => {
        assignDagre();
        resetCamera(sigma);
    };

    const runStandardLayout = (): void => {
        random.assign(graph, { scale: 1000 });
        forceAtlas2.assign(graph, {
            iterations: 128,
            settings: {
                scalingRatio: 1000,
                barnesHutOptimize: true,
            },
        });
        resetCamera(sigma);
    };

    const reset = (): void => {
        resetCamera(sigma);
    };

    return (
        <Box position={'absolute'} bottom={16} display={'flex'}>
            <GraphButton onClick={reset} displayText={<FontAwesomeIcon icon={faCropAlt} />} />
            {sequential && <GraphButton onClick={runSequentialLayout} displayText='sequential' />}
            {standard && <GraphButton onClick={runStandardLayout} displayText='standard' />}
            {nonLayoutButtons?.length && (
                <>
                    {nonLayoutButtons.map((props, index) => (
                        <GraphButton key={index} onClick={props.onClick} displayText={props.displayText} />
                    ))}
                </>
            )}
        </Box>
    );
};

export default GraphButtons;
