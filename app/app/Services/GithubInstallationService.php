<?php

namespace App\Services;

use App\Models\Installation;
use App\Models\Repository;
use Firebase\JWT\JWT;
use Illuminate\Support\Facades\Http;
use Illuminate\Support\Facades\Storage;

class GithubInstallationService
{

    protected Installation $installation;

    public function __construct(Installation $installation)
    {
        $this->installation = $installation;

        if ($this->isAccessTokenExpired()) {
            $this->refreshAccessToken();
        }
    }

    public static function getAccessToken(string $installationId, $JWTWebToken)
    {
        $response = Http::withHeaders([
            'Authorization' => 'Bearer ' . $JWTWebToken,
            'Accept' => 'application/vnd.github.v3+json'
        ])->post("https://api.github.com/app/installations/{$installationId}/access_tokens");

        return $response->json();
    }

    public function isAccessTokenExpired()
    {
        return now()->gte($this->installation->expires_at);
    }

    public function refreshAccessToken()
    {
        $response = Http::withHeaders([
            'Authorization' => 'Bearer ' . $this->generateJWTWebToken(),
            'Accept' => 'application/vnd.github.v3+json'
        ])->post("https://api.github.com/app/installations/{$this->installation->installation_id}/access_tokens");


        $data = $response->json();

        Installation::where('installation_id', $this->installation->installation_id)->update([
            'access_token' => $data['token'],
            'expires_at' => $data['expires_at'],
            'repository_selection' => $data['repository_selection']
        ]);
    }

    public static function generateJWTWebToken()
    {
        $clientId = env('GITHUB_CLIENT_ID');
        $privateKey = Storage::get(env("GITHUB_PRIVATE_KEY"));
        $algorithm = 'RS256';

        $time = time();

        $payload = [
            'iat' => $time,
            'exp' => $time + 60,
            'iss' => $clientId
        ];

        return JWT::encode($payload, $privateKey, $algorithm);
    }

    public function fetchRepositories()
    {
        $response = Http::withHeaders([
            'Authorization' => 'Bearer ' . $this->installation->access_token,
            'Accept' => 'application/vnd.github.v3+json'
        ])->get("https://api.github.com/installation/repositories");

        return $response->json();
    }

    public function fetchWorkflows(string $repository)
    {
        $response = Http::withHeaders([
            'Authorization' => 'Bearer ' . $this->installation->access_token,
            'Accept' => 'application/vnd.github.v3+json'
        ])->get("https://api.github.com/repos/{$repository}/actions/workflows");

        return $response->json();
    }
}
