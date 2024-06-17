<?php

namespace App\Actions\Github;

use App\Models\Session;
use App\Services\GithubInstallationService;
use Illuminate\Support\Arr;

class DispatchSessionRun
{
    public function handle(Session $session, array $data = []): array
    {
        $client = new GithubInstallationService($session->installation);

        $workflows = collect($session->data)
            ->map(fn($suite) => $this->dispatchWorkflow(
                client: $client,
                suite: $suite,
                data: $data
            ))
            ->toArray();

        // TODO: Save runs to the DB
        // Dispatching a workflow doesn't return the run ID

        /**
         collect($workflows)->each(fn ($workflow) => $session->runs()->create([
            'github_run_id' => Arr::get($workflow, 'id'),
            ...
         ]));
         */

        return $workflows;
    }

    protected function dispatchWorkflow(GithubInstallationService $client, array $suite, array $data)
    {
        return $client->dispatchWorkflow(
            repository: Arr::get($suite, 'repository_name'),
            workflowId: Arr::get($suite, 'workflow.id'),
            inputs: Arr::get($data, 'inputs', [])
        );
    }
}
