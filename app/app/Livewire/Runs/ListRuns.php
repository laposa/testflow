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

    public function mounted(Session $session)  {
        $this->session = $session;
    }

    public function render(FetchSessionWorkflowRuns $fetchSessionWorkflowRuns)
    {
        $this->session->load('items.runs');
        $pollingEnabled = !$fetchSessionWorkflowRuns->handle($this->session);

        return view('livewire.runs.list-runs', [
            'session' => $this->session,
            'pollingEnabled' => $pollingEnabled,
        ]);
    }

    public function createRun(DispatchSessionRun $dispatchSessionRun)
    {
        $dispatchSessionRun->handle($this->session);
    }
}
