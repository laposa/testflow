<?php

namespace App\Actions\Github;

use App\Models\Installation;
use App\Services\GithubClient;

class InstallGithubApp
{
    public function handle($installationId, $action): Installation
    {
        $data = GithubClient::getAccessToken($installationId, GithubClient::generateJWTWebToken());

        return Installation::updateOrCreate(
            [
                'installation_id' => $installationId,
            ],
            [
                'access_token' => $data['token'],
                'expires_at' => $data['expires_at'],
                'repository_selection' => $data['repository_selection'],
            ],
        );
    }
}
