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

        return view('livewire.runs.list-runs', [
            'session' => $this->session,
            'pollingEnabled' => $pollingEnabled,
        ]);
    }

    public function createRun(DispatchSessionRun $dispatchSessionRun)
    {
        $dispatchSessionRun->handle($this->session, null);
        $this->dispatch('reload-activity');
    }

    public function rerunRun(DispatchSessionRun $dispatchSessionRun, $runId)
    {
        $dispatchSessionRun->handle(
            $this->session,
            $this->session->runs()->find($runId)->items->toArray(),
        );
        $this->dispatch('reload-activity');
    }
}
