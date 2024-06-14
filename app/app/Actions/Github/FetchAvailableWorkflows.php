<?php

namespace App\Actions\Github;

use App\Models\Installation;
use App\Models\Repository;
use App\Services\GithubInstallationService;

class FetchAvailableWorkflows
{
    public function handle(Installation $installation) {
        $client = new GithubInstallationService($installation);
        $data = Repository::all()->map(function(Repository $repository) use ($client) {
            $workflows = $client->fetchWorkflows($repository->full_name);
            return [
                'repository' => $repository,
                'workflows' => $workflows
            ];
        });

        return $data;
    }
}
