<?php

namespace App\Actions\Github;

use App\Models\Installation;
use App\Services\GithubInstallationService;
use Illuminate\Support\Arr;

class FetchTestSuites
{
    protected GithubInstallationService $client;

    public function __construct()
    {
        $installation = Installation::first();
        $this->client = new GithubInstallationService($installation);
    }

    public function handle(): array
    {
        $repositories = $this->client->fetchRepositories();
        $suites = [];

        collect(Arr::get($repositories, 'repositories', []))
            ->each(function ($repository) use (&$suites) {
                $contents = $this->fetchContents($repository);

                if ($contents) {
                    $workflows = $this->fetchWorkflows($repository);
                    array_push(
                        $suites,
                        ...$this->formatSuites(
                            contents: $contents,
                            repository: $repository,
                            workflows: $workflows
                        )
                    );
                }
            });

        return $suites;
    }

    protected function formatSuites($contents, $repository, $workflows)
    {
        return collect($contents)
            ->map(fn ($directory) => [
                "repository_id" => $repository['id'],
                "repository_name" => $repository['full_name'],
                "path" => "{$repository['full_name']}/{$directory['path']}",
                "workflow" => collect(Arr::get($workflows, 'workflows', []))
                    // TODO: Filter by workflow name?
                    ->filter(fn ($workflow) => $workflow['name'] === "Run integration tests")
                    ->first()
            ]
            )
            ->toArray();
    }

    protected function fetchContents($repository) {
        try {
            return $this->client->fetchRepositoryContents(
                fullName: $repository['full_name'],
                path: 'tests'
            );
        } catch (\Exception $e) {
            return null;
        }
    }

    protected function fetchWorkflows($repository) {
        try {
            return $this->client->fetchWorkflows($repository['full_name']);
        } catch (\Exception $e) {
            return null;
        }
    }
}
