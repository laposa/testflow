<?php

namespace App\Actions\Github;

use App\Models\Session;
use App\Services\GithubInstallationService;
use Illuminate\Support\Arr;

class FetchSessionWorkflowRuns
{

    public function handle(Session $session): array
    {
        return collect($session->data)
                ->map(fn ($data) => $this->fetchWorkflowRuns(
                    session: $session,
                    data: $data
                ))
                ->toArray();
    }

    protected function fetchWorkflowRuns(Session $session, array $data) : array
    {
        $client = new GithubInstallationService($session->installation);

        return $client->fetchWorkflowRuns(
            repository: Arr::get($data, 'repository_name', null),
            workflowId: Arr::get($data, 'workflow.id', [])
        )['workflow_runs'];
    }
}
