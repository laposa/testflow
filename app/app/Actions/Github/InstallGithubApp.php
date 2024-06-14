<?php

namespace App\Actions\Github;

use App\Models\Installation;
use App\Models\Repository;
use App\Services\GithubInstallationService;
use Firebase\JWT\JWT;
use Illuminate\Support\Facades\Http;
use Illuminate\Support\Facades\Storage;

class InstallGithubApp
{
    public function handle($installationId, $action): Installation
    {
        $data = GithubInstallationService::getAccessToken($installationId, GithubInstallationService::generateJWTWebToken());

        $installation = Installation::updateOrCreate([
            'installation_id' => $installationId
        ], [
            'access_token' => $data['token'],
            'expires_at' => $data['expires_at'],
            'repository_selection' => $data['repository_selection']
        ]);

        app(SyncRepositories::class)->handle($installation);

        return $installation;
    }




}
