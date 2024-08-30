<?php

namespace App\Livewire\Runs;

use App\Actions\Github\DispatchSessionRun;
use App\Actions\Github\FetchSessionWorkflowRuns;
use App\Models\Session;
use Livewire\Component;

class ListRuns extends Component
{
    public Session $session;

    public bool $isLoading = false;

    public function mounted(Session $session)
    {
        $this->session = $session;
    }

    public function render(FetchSessionWorkflowRuns $fetchSessionWorkflowRuns)
    {
        $this->session->load('items.runs');
        $this->session->load([
            'runs' => function ($query) {
                $query->orderBy('created_at', 'desc');
            },
        ]);
        $pollingEnabled = !$fetchSessionWorkflowRuns->handle($this->session);

        $failedRuns = $this->getFailedRuns();
        return view('livewire.runs.list-runs', [
            'session' => $this->session,
            'pollingEnabled' => $pollingEnabled,
            'failedRuns' => $failedRuns,
        ]);
    }

    public function createRun(DispatchSessionRun $dispatchSessionRun)
    {
        $dispatchSessionRun->handle($this->session, null);
        $this->dispatch('reload-activity');
    }

    public function rerunFailed(DispatchSessionRun $dispatchSessionRun)
    {
        $items = $this->getFailedRuns()
            ->map(fn($run) => $run->items->toArray())
            ->flatten(1)
            ->toArray();

        $dispatchSessionRun->handle($this->session, $items);
        $this->dispatch('reload-activity');
    }

    protected function getFailedRuns()
    {
        $failedRuns = [];
        $successfulRuns = [];
        foreach ($this->session->runs as $run) {
            $groupBy = $run->repository_name . $run->service_name;
            // skip runs that have already passed
            if ($run->parsedResults->getTotalFailures() > 0 && !isset($successfulRuns[$groupBy])) {
                $failedRuns[$groupBy] = $run;
            }

            if ($run->parsedResults->getTotalFailures() === 0) {
                $successfulRuns[$groupBy] = $run;
            }
        }

        return collect($failedRuns);
    }
}
