<?php

namespace App\Actions\Github;

use App\Models\Installation;
use App\Models\Repository;
use App\Services\GithubInstallationService;

class SyncRepositories {
    public function handle(Installation $installation) {
        $client = new GithubInstallationService($installation);
        $data = $client->fetchRepositories();

        collect($data['repositories'])->each(function ($repository) {
            Repository::updateOrCreate([
                'repository_id' => $repository['id']
            ], [
                'name' => $repository['name'],
                'full_name' => $repository['full_name'],
            ]);
        });

        Repository::whereNotIn('repository_id', collect($data['repositories'])->pluck('id'))->delete();   // Fetch all repositories from the Github API
    }
}
