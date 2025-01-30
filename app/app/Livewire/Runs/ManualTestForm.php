<?php

namespace App\Livewire\Runs;

use App\Actions\Github\DispatchSessionRun;
use App\Actions\Github\FetchSessionWorkflowRuns;
use App\Models\Session;
use App\Models\SessionServiceRun;
use App\Models\SessionServiceSuiteTest;
use Illuminate\Support\Collection;
use Livewire\Component;
use Livewire\Attributes\On;

class ManualTestForm extends Component
{
    public SessionServiceSuiteTest $test;
    public SessionServiceRun $run;
    public $index;

    public string $result = '';
    public string $comment = "";
    public string $error = "";

    public function save()
    {
        $this->dispatch('updated',
            id: $this->test->id,
            result: $this->result,
            comment: $this->comment
        );
    }

    public function skip()
    {
        $this->dispatch('updated',
            id: $this->test->id,
            result: 'skipped',
            comment: $this->comment
        );
    }

    public function initializeValues()
    {
        $results = getResultsFromXML($this->run);

        $this->error = validateManualTest($this->test);
        $this->comment = $results[$this->test->id]["comment"] ?? "";
        $this->result = $results[$this->test->id]["status"] ?? "";
    }

    public function render()
    {
        $this->initializeValues();
        
        return view('livewire.runs.manual-test-form');
    }
}
