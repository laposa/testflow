<?php

namespace App\Livewire\Runs;

use App\Actions\Github\FetchSessionWorkflowRuns;
use App\Models\Session;
use Livewire\Component;

class ListRuns extends Component
{
    public Session $session;

    public function mounted(Session $session)  {
        $this->session = $session;
    }

    public function render(FetchSessionWorkflowRuns $fetchSessionWorkflowRuns)
    {
        $this->session->load('items.runs');
        $fetchSessionWorkflowRuns->handle($this->session);
        $this->session->load('items.runs');

        return view('livewire.runs.list-runs', [
            'session' => $this->session
        ]);
    }
}
