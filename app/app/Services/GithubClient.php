<?php

namespace App\Services;

use App\Models\Installation;
use Firebase\JWT\JWT;
use Illuminate\Http\Client\PendingRequest;
use Illuminate\Support\Facades\Http;

class GithubClient
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
            'Accept' => 'application/vnd.github.v3+json',
        ])->baseUrl($this->baseUrl);
    }

    public static function getAccessToken(string $installationId, $JWTWebToken)
    {
        $response = Http::withHeaders([
            'Authorization' => 'Bearer ' . $JWTWebToken,
            'Accept' => 'application/vnd.github.v3+json',
        ])->post("https://api.github.com/app/installations/{$installationId}/access_tokens");

        return $response->json();
    }

    public function isAccessTokenExpired(): bool
    {
        return now()->gte($this->installation->expires_at);
    }

    public function refreshAccessToken(): void
    {
        $response = Http::withHeaders([
            'Authorization' => 'Bearer ' . $this->generateJWTWebToken(),
            'Accept' => 'application/vnd.github.v3+json',
        ])->post(
            "https://api.github.com/app/installations/{$this->installation->installation_id}/access_tokens",
        );

        $data = $response->json();

        $this->installation->update([
            'access_token' => $data['token'],
            'expires_at' => $data['expires_at'],
            'repository_selection' => $data['repository_selection'],
        ]);
    }

    public static function generateJWTWebToken(): string
    {
        $clientId = env('GITHUB_CLIENT_ID');
        $privateKey = env('GITHUB_PRIVATE_KEY');
        $algorithm = 'RS256';

        $time = time();

        $payload = [
            'iat' => $time,
            'exp' => $time + 60,
            'iss' => $clientId,
        ];

        return JWT::encode($payload, $privateKey, $algorithm);
    }

    public function fetchRepositories()
    {
        $response = $this->withHeaders()->get('/installation/repositories');

        return $response->json();
    }

    public function fetchRepositoryContents($fullName, $path, $ref = null)
    {
        $url = "/repos/{$fullName}/contents/{$path}";
        if ($ref) {
            $url .= "?ref={$ref}";
        }

        $response = $this->withHeaders()->get($url);

        return $response->json();
    }

    public function fetchWorkflows(string $repository)
    {
        $response = $this->withHeaders()->get("/repos/{$repository}/actions/workflows");

        return $response->json();
    }

    public function fetchWorkflowRuns(
        string $repository,
        string $workflowId,
        $perPage = 30,
        $page = 1,
    ) {
        $response = $this->withHeaders()->get(
            "/repos/{$repository}/actions/workflows/{$workflowId}/runs?per_page={$perPage}&page={$page}",
        );

        return $response->json();
    }

    public function fetchAllWorkflowRuns(string $repository, string $workflowId)
    {
        $page = 1;
        $perPage = 100;
        $workflowRuns = [];

        do {
            $response = $this->fetchWorkflowRuns($repository, $workflowId, $perPage, $page);

            $workflowRuns = array_merge($workflowRuns, $response['workflow_runs']);

            $page++;
        } while (count($response['workflow_runs']) === $perPage);

        return $workflowRuns;
    }

    public function dispatchWorkflow(string $repository, string $workflowId, array $inputs)
    {
        $response = $this->withHeaders()
            ->post("/repos/{$repository}/actions/workflows/{$workflowId}/dispatches", [
                'ref' => 'master',
                'inputs' => [
                    'environment' => 'preprod',
                    ...$inputs,
                ],
            ])
            ->throw();

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

    public function fetchWorkflowRunArtifacts(string $repository, string $runId)
    {
        $response = $this->withHeaders()->get(
            "/repos/{$repository}/actions/runs/{$runId}/artifacts",
        );

        return $response->json();
    }



    public function fetchWorkflowRunArtifactsDownload(string $repository, string $artifactId)
    {
        $response = $this->withHeaders()->get(
            "/repos/{$repository}/actions/artifacts/{$artifactId}/zip",
        );

        return $response->body();
    }

    public function fetchWorkflowRunJobs(string $repository, string $runId)
    {
        $response = $this->withHeaders()->get("/repos/{$repository}/actions/runs/{$runId}/jobs");

        return $response->json();
    }

    public function fetchFileContents(string $path)
    {
//        dd("/repos/contents$path");
        $response = $this->withHeaders()->get("/repos/contents$path");
        return $response->json();
    }
}
