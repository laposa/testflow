<?php

namespace App\Livewire\Runs;

use App\Actions\Github\DispatchSessionRun;
use App\Actions\Github\FetchSessionWorkflowRuns;
use App\Models\Session;
use App\Models\SessionServiceRun;
use App\Models\SessionServiceSuiteTest;
use Illuminate\Support\Collection;
use Livewire\Component;

class ManualTestForm extends Component
{
    public SessionServiceSuiteTest $test;

    public string $result = 'fail';
    public string $comment = "";

    public function save()
    {
        $this->dispatch('test-updated', $this->test->id, $this->result, $this->comment);
    }

    public function render()
    {
        return view('livewire.runs.manual-test-form');
    }


}
