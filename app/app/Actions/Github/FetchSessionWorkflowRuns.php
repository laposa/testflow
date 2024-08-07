<?php

namespace App\Actions\Github;

use App\Models\Session;
use App\Models\SessionItem;
use App\Services\GithubInstallationService;
use Illuminate\Support\Facades\Storage;

class FetchSessionWorkflowRuns
{
    protected $workflowsUpdated = false;

    public function handle(Session $session): bool
    {
        $this->workflowsUpdated = false;
        collect($session->items)
            ->groupBy('workflow_id')
            ->map(fn($items) => $this->fetchRunsForItem($session, $items[0]));

        if ($this->workflowsUpdated) {
            $session->load('items.runs');
        }

        // if all runs have logs in the DB, return true
        return $session->items->every(
            fn($item) => $item->runs->every(fn($run) => $run->result_log),
        );
    }

    protected function fetchRunsForItem(Session $session, SessionItem $item)
    {
        $client = new GithubInstallationService($session->installation);

        // if all runs have logs in the DB, skip fetching from github
        if ($item->runs->every(fn($run) => $run->run_log)) {
            return $item->runs;
        }

        // Github Status: queued, in_progress, completed
        // Github Conclusion: success, failure, neutral, cancelled, skipped, timed_out, action_required

        // update item's runs with the latest data from github (based on the crated_at timestamp +- 5 seconds)
        $workflowRuns = $client->fetchWorkflowRuns($item->repository_name, $item->workflow_id)[
            'workflow_runs'
        ];
        foreach ($item->runs as $run) {
            $runData = collect($workflowRuns)->first(
                fn($workflowRun) => $workflowRun['created_at'] >=
                    $run->created_at->subSeconds(5)->toIso8601String() &&
                    $workflowRun['created_at'] <=
                        $run->created_at->addSeconds(5)->toIso8601String(),
            );

            if ($runData) {
                $this->workflowsUpdated = true;

                $run->update([
                    'status' => $runData['conclusion']
                        ? $runData['conclusion']
                        : $runData['status'],
                ]);

                if ($runData['status'] === 'completed') {
                    if (!$run->result_log) {
                        $run->result_log = $this->fetchWorkflowResultsLog(
                            $client,
                            $item,
                            $runData['id'],
                        );
                    }

                    if (!$run->run_log) {
                        $run->run_log = $this->fetchWorkflowRunLog($client, $item, $runData['id']);
                    }

                    $run->save();
                }
            }
        }
    }

    protected function fetchWorkflowRunLog(
        GithubInstallationService $client,
        SessionItem $item,
        $runId,
    ) {
        $zipFile = $client->fetchWorkflowRunLog($item->repository_name, $runId);

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
        SessionItem $item,
        $runId,
    ) {
        // find artifact with the name 'test-results'
        $artifacts = $client->fetchWorkflowRunArtifacts($item->repository_name, $runId);
        $artifact = collect($artifacts['artifacts'])->first(
            fn($artifact) => $artifact['name'] === 'test-results',
        );

        if (!$artifact) {
            return null;
        }

        $zipFile = $client->fetchWorkflowRunArtifactsDownload(
            $item->repository_name,
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
