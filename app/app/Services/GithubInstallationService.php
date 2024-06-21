<?php

namespace App\Services;

use App\Models\Installation;
use Firebase\JWT\JWT;
use Illuminate\Http\Client\PendingRequest;
use Illuminate\Support\Facades\Http;

class GithubInstallationService
{

    protected Installation $installation;
    protected string $baseUrl = 'https://api.github.com';

    public function __construct(Installation $installation)
    {
        $this->installation = $installation;

        if ($this->isAccessTokenExpired()) {
            $this->refreshAccessToken();
        }
    }


    public function withHeaders(): PendingRequest
    {
        return Http::withHeaders([
            'Authorization' => 'Bearer ' . $this->installation->access_token,
            'Accept' => 'application/vnd.github.v3+json'
        ])->baseUrl($this->baseUrl);
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

        $this->installation->update([
            'access_token' => $data['token'],
            'expires_at' => $data['expires_at'],
            'repository_selection' => $data['repository_selection']
        ]);
    }

    public static function generateJWTWebToken()
    {
        $clientId = env('GITHUB_CLIENT_ID');
        $privateKey = env("GITHUB_PRIVATE_KEY");
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
        $response = $this->withHeaders()->get("/installation/repositories");

        return $response->json();
    }

    public function fetchRepositoryContents($fullName, $path)
    {
        $response = $this->withHeaders()->get("/repos/{$fullName}/contents/{$path}");

        return $response->json();
    }

    public function fetchWorkflows(string $repository)
    {
        $response = $this->withHeaders()->get("/repos/{$repository}/actions/workflows");

        return $response->json();
    }

    public function fetchWorkflowRuns(string $repository, string $workflowId)
    {
        $response = $this->withHeaders()->get("/repos/{$repository}/actions/workflows/{$workflowId}/runs");

        return $response->json();
    }

    public function dispatchWorkflow(string $repository, string $workflowId, array $inputs)
    {
        $response = $this->withHeaders()->post("/repos/{$repository}/actions/workflows/{$workflowId}/dispatches", [
            'ref' => 'master',
            'inputs' => [
                'environment' => "preprod"
            ]
        ])->throw();


        ray("RESPONSE", $response->json());
        return $response->json();
    }

    public function fetchLatestCommit(string $repository, string $branch = 'master')
    {
        $response = $this->withHeaders()->get("/repos/{$repository}/commits/{$branch}", [
            'per_page' => 1,
        ]);
        return $response->json();
    }

    public function fetchWorkflowRunLog(string $repository, string $runId)
    {
        $response = $this->withHeaders()->get("/repos/{$repository}/actions/runs/{$runId}/logs");
        return $response->body();
    }
}
