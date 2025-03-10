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

import { useState } from 'react';
import { useAppDispatch } from 'src/store';
import { Box, Button, CircularProgress, Grid, Switch, Typography } from '@mui/material';
import { PageWithTitle, Enable2FADialog, Disable2FADialog } from 'bh-shared-ui';
import TextWithFallback from 'src/components/TextWithFallback';
import PasswordDialog from '../Users/PasswordDialog';
import { getUsername } from 'src/utils';
import { useMutation, useQuery } from 'react-query';
import apiClient from 'src/api';
import { addSnackbar } from 'src/ducks/global/actions';
import { Alert, AlertTitle } from '@mui/material';

const UserProfile = () => {
    const dispatch = useAppDispatch();
    const [changePasswordDialogOpen, setChangePasswordDialogOpen] = useState(false);
    const [enable2FADialogOpen, setEnable2FADialogOpen] = useState(false);
    const [disable2FADialogOpen, setDisable2FADialogOpen] = useState(false);
    const [TOTPSecret, setTOTPSecret] = useState('');
    const [QRCode, setQRCode] = useState('');
    const [enable2FAError, setEnable2FAError] = useState('');
    const [disable2FAError, setDisable2FAError] = useState('');

    const getSelfQuery = useQuery(['getSelf'], ({ signal }) =>
        apiClient.getSelf({ signal }).then((res) => res.data.data)
    );

    const updateUserPasswordMutation = useMutation(
        ({ userId, secret, needsPasswordReset }: { userId: string; secret: string; needsPasswordReset: boolean }) =>
            apiClient.putUserAuthSecret(userId, {
                needs_password_reset: needsPasswordReset,
                secret: secret,
            }),
        {
            onSuccess: () => {
                dispatch(addSnackbar('Password updated successfully!', 'updateUserPasswordSuccess'));
                setChangePasswordDialogOpen(false);
            },
        }
    );

    if (getSelfQuery.isLoading) {
        return (
            <PageWithTitle title='My Profile' data-testid='my-profile'>
                <Typography variant='h2'>User Information</Typography>
                <Box p={4} textAlign='center'>
                    <CircularProgress />
                </Box>
            </PageWithTitle>
        );
    }

    if (getSelfQuery.isError) {
        return (
            <PageWithTitle title='My Profile' data-testid='my-profile'>
                <Typography variant='h2'>User Information</Typography>

                <Alert severity='error'>
                    <AlertTitle>Error</AlertTitle>
                    Sorry, there was a problem fetching your user information.
                    <br />
                    Please try refreshing the page or logging in again.
                </Alert>
            </PageWithTitle>
        );
    }

    const user = getSelfQuery.data;

    return (
        <>
            <PageWithTitle title='My Profile' data-testid='my-profile'>
                <Typography variant='h2'>User Information</Typography>

                <Grid container spacing={2} alignItems='center'>
                    <Grid item xs={3}>
                        <Typography variant='body1'>Email</Typography>
                    </Grid>
                    <Grid item xs={9}>
                        <Typography variant='body1'>{user?.email_address}</Typography>
                    </Grid>

                    <Grid item xs={3}>
                        <Typography variant='body1'>Name</Typography>
                    </Grid>
                    <Grid item xs={9}>
                        <Typography variant='body1'>
                            <TextWithFallback text={getUsername(user)} fallback='Unknown' />
                        </Typography>
                    </Grid>

                    <Grid item xs={3}>
                        <Typography variant='body1'>Role</Typography>
                    </Grid>
                    <Grid item xs={9}>
                        <Typography variant='body1'>
                            <TextWithFallback text={user?.roles?.[0]?.name} fallback='Unknown' />
                        </Typography>
                    </Grid>
                </Grid>
                {user.saml_provider_id === null && (
                    <>
                        <Box mt={2}>
                            <Typography variant='h2'>Authentication</Typography>
                        </Box>
                        <Grid container spacing={2} alignItems='center'>
                            <Grid item xs={3}>
                                <Typography variant='body1'>Password</Typography>
                            </Grid>
                            <Grid item xs={9}>
                                <Button
                                    variant='contained'
                                    color='primary'
                                    size='small'
                                    disableElevation
                                    onClick={() => setChangePasswordDialogOpen(true)}
                                    data-testid='my-profile_button-reset-password'>
                                    Reset Password
                                </Button>
                            </Grid>

                            <Grid item xs={3}>
                                <Typography variant='body1'>Two-Factor Authentication</Typography>
                            </Grid>
                            <Grid item xs={9}>
                                <Box display='flex' alignItems='center'>
                                    <Switch
                                        inputProps={{
                                            'aria-label': 'Two-Factor Authentication Enabled',
                                        }}
                                        checked={user.AuthSecret?.totp_activated}
                                        onChange={() => {
                                            if (!user.AuthSecret?.totp_activated) setEnable2FADialogOpen(true);
                                            else setDisable2FADialogOpen(true);
                                        }}
                                        color='primary'
                                        data-testid='my-profile_switch-two-factor-authentication'
                                    />
                                    {user.AuthSecret?.totp_activated && (
                                        <Typography variant='body1'>Enabled</Typography>
                                    )}
                                </Box>
                            </Grid>
                        </Grid>
                    </>
                )}
            </PageWithTitle>

            <PasswordDialog
                open={changePasswordDialogOpen}
                onClose={() => setChangePasswordDialogOpen(false)}
                userId={user.id}
                showNeedsPasswordReset={false}
                onSave={updateUserPasswordMutation.mutate}
            />

            <Enable2FADialog
                open={enable2FADialogOpen}
                onClose={() => {
                    setEnable2FADialogOpen(false);
                    setEnable2FAError('');
                    getSelfQuery.refetch();
                }}
                onCancel={() => {
                    setEnable2FADialogOpen(false);
                    setEnable2FAError('');
                    getSelfQuery.refetch();
                }}
                onSavePassword={(password) => {
                    setEnable2FAError('');
                    return apiClient
                        .enrollMFA(user.id, {
                            secret: password,
                        })
                        .then((response) => {
                            setQRCode(response.data.data.qr_code);
                            setTOTPSecret(response.data.data.totp_secret);
                            setEnable2FAError('');
                        })
                        .catch((err) => {
                            setEnable2FAError('Unable to verify password. Please try again.');
                            throw err;
                        });
                }}
                onSaveOTP={(OTP) => {
                    setEnable2FAError('');
                    return apiClient
                        .activateMFA(user.id, {
                            otp: OTP,
                        })
                        .then(() => {
                            setEnable2FAError('');
                        })
                        .catch((err) => {
                            setEnable2FAError('Unable to verify one-time password. Please try again.');
                            throw err;
                        });
                }}
                onSave={() => {
                    setEnable2FADialogOpen(false);
                    setEnable2FAError('');
                    getSelfQuery.refetch();
                }}
                TOTPSecret={TOTPSecret}
                QRCode={QRCode}
                error={enable2FAError}
            />

            <Disable2FADialog
                open={disable2FADialogOpen}
                onClose={() => {
                    setDisable2FADialogOpen(false);
                    setDisable2FAError('');
                    getSelfQuery.refetch();
                }}
                onCancel={() => {
                    setDisable2FADialogOpen(false);
                    setDisable2FAError('');
                    getSelfQuery.refetch();
                }}
                onSave={(secret: string) => {
                    setDisable2FAError('');
                    apiClient
                        .disenrollMFA(user.id, { secret })
                        .then(() => {
                            setDisable2FADialogOpen(false);
                            setDisable2FAError('');
                            getSelfQuery.refetch();
                        })
                        .catch(() => {
                            setDisable2FAError('Unable to verify password. Please try again.');
                        });
                }}
                error={disable2FAError}
            />
        </>
    );
};

export default UserProfile;
