<?php

namespace App\Console\Commands;

use App\Models\SessionServiceRun;
use Illuminate\Console\Command;
use App\Services\GithubClient;

class SyncGithubRuns extends Command
{
    /**
     * The name and signature of the console command.
     *
     * @var string
     */
    protected $signature = 'app:sync-github-runs';

    /**
     * The console command description.
     *
     * @var string
     */
    protected $description = 'Will find the github run for each service run based on the created_at timestamp and update the database with the github data';

    /**
     * Execute the console command.
     */
    public function handle()
    {
        $runs = SessionServiceRun::select([
            'id',
            'service_id',
            'created_at',
            'started_at',
            'finished_at',
            'commit_sha',
            'github_run_id',
        ])
            ->with('service')
            ->get();

        $client = new GithubClient();

        $workflowRuns = [];

        foreach ($runs as $run) {
            $id = $run->service->repository_name . '_' . $run->service->workflow_id;

            if (!isset($workflowRuns[$id])) {
                $workflowRuns[$id] = $client->fetchAllWorkflowRuns(
                    $run->service->repository_name,
                    $run->service->workflow_id,
                );
            }

            // find the workflow run based no the created_at timestamp +- 5 seconds
            $runData = collect($workflowRuns[$id])->first(
                fn($workflowRun) => $workflowRun['created_at'] >=
                    $run->created_at->subSeconds(5)->toIso8601String() &&
                    $workflowRun['created_at'] <=
                        $run->created_at->addSeconds(5)->toIso8601String(),
            );

            if ($runData) {
                $this->info("Updating run {$run->id} with github_run_id: {$runData['id']}");

                $jobs = $client->fetchWorkflowRunJobs(
                    $run->service->repository_name,
                    $runData['id'],
                )['jobs'];

                $startedAt = collect($jobs)->min('started_at');
                $finishedAt = collect($jobs)->max('completed_at');

                $run->update([
                    'github_run_id' => $runData['id'],
                    'commit_sha' => $runData['head_sha'],
                    'started_at' => $startedAt,
                    'finished_at' => $finishedAt,
                ]);
            }
        }
    }
}
