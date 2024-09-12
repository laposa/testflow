<?php

namespace App\Actions\Github;

use App\Models\Session;
use App\Models\SessionService;
use App\Models\SessionServiceRun;
use App\Services\GithubInstallationService;
use Illuminate\Support\Facades\Storage;
use App\Enums\SessionActivityType;

class FetchSessionWorkflowRuns
{
    protected $workflowsUpdated = false;

    public function handle(Session $session): bool
    {
        $this->workflowsUpdated = false;
        $session->services->map(fn($service) => $this->fetchRunsForService($session, $service));

        return $this->workflowsUpdated;
    }

    protected function fetchRunsForService(Session $session, SessionService $service)
    {
        $client = new GithubInstallationService($session->installation);

        // if all runs have logs in the DB, skip fetching from github
        if ($service->runs->every(fn($run) => $run->run_log)) {
            return;
        }

        // Github Status: queued, in_progress, completed
        // Github Conclusion: success, failure, neutral, cancelled, skipped, timed_out, action_required

        // update item's runs with the latest data from github (based on the crated_at timestamp +- 5 seconds)
        $workflowRuns = $client->fetchWorkflowRuns(
            $service->repository_name,
            $service->workflow_id,
        )['workflow_runs'];

        foreach ($service->runs as $run) {
            // find the workflow run based no the created_at timestamp +- 5 seconds
            $runData = collect($workflowRuns)->first(
                fn($workflowRun) => $workflowRun['created_at'] >=
                    $run->created_at->subSeconds(5)->toIso8601String() &&
                    $workflowRun['created_at'] <=
                        $run->created_at->addSeconds(5)->toIso8601String(),
            );

            if ($runData) {
                $this->workflowsUpdated = true;

                $oldStatus = $run->status;

                $run->update([
                    'status' => $runData['conclusion']
                        ? $runData['conclusion']
                        : $runData['status'],
                ]);

                if ($runData['status'] === 'completed') {
                    if (!$run->result_log) {
                        $run->result_log = $this->fetchWorkflowResultsLog(
                            $client,
                            $service,
                            $runData['id'],
                        );

                        $run->passed = $run->parsedResults->getTotalPassed();
                        $run->failed = $run->parsedResults->getTotalFailures();
                        $run->duration = $run->parsedResults->getTotalDuration();
                    }

                    if (!$run->run_log) {
                        $run->run_log = $this->fetchWorkflowRunLog(
                            $client,
                            $service,
                            $runData['id'],
                        );
                    }

                    $run->save();
                }

                $this->logRunStatusChange($session, $run, $oldStatus);
            }
        }
    }

    protected function logRunStatusChange(Session $session, SessionServiceRun $run, $oldStatus)
    {
        if ($oldStatus === $run->status) {
            return;
        }

        $body = "Run {$run->id} for {$run->service->displayName} has changed status from {$oldStatus} to {$run->status}";
        if ($run->result_log) {
            $route = route('session.runs.show', [$session, $run], false);
            $body = str_replace($run->id, "<a href='{$route}'>{$run->id}</a>", $body);
            $body .= "  with <span class='pass'>{$run->passed} passed</span> and <span class='fail'>{$run->failed} failed</span> tests";
        }
        $body .= '.';

        $session->activity()->create([
            'type' => SessionActivityType::run_status_changed,
            'body' => $body,
            'user_id' => auth()->id(),
        ]);
    }

    protected function fetchWorkflowRunLog(
        GithubInstallationService $client,
        SessionService $service,
        $runId,
    ) {
        $zipFile = $client->fetchWorkflowRunLog($service->repository_name, $runId);

        $tmpFileName = 'github_run_logs_' . $runId . '_' . time() . '.zip';
        Storage::put($tmpFileName, $zipFile);

        // unzip the zipFile and find the largest file in the zip
        // ie the log file combining all the logs from the workflow run
        $zip = new \ZipArchive();
        $zip->open(storage_path('app/' . $tmpFileName));

        $log = '';
        $size = 0;

        for ($i = 0; $i < $zip->numFiles; $i++) {
            $stat = $zip->statIndex($i);
            if ($stat['size'] > $size) {
                $size = $stat['size'];
                $log = $zip->getFromIndex($i);
            }
        }
        $zip->close();

        Storage::delete($tmpFileName);

        return $log;
    }

    protected function fetchWorkflowResultsLog(
        GithubInstallationService $client,
        SessionService $service,
        $runId,
    ) {
        // find artifact with the name 'test-results'
        $artifacts = $client->fetchWorkflowRunArtifacts($service->repository_name, $runId);
        $artifact = collect($artifacts['artifacts'])->first(
            fn($artifact) => $artifact['name'] === 'test-results',
        );

        if (!$artifact) {
            return null;
        }

        $zipFile = $client->fetchWorkflowRunArtifactsDownload(
            $service->repository_name,
            $artifact['id'],
        );

        $tmpFileName = 'github_result_logs_' . $runId . '_' . time() . '.zip';
        Storage::put($tmpFileName, $zipFile);

        // unzip the zipFile and get the first file in the zip
        $zip = new \ZipArchive();
        $zip->open(storage_path('app/' . $tmpFileName));

        $log = $zip->getFromIndex(0);

        Storage::delete($tmpFileName);

        return $log;
    }
}
