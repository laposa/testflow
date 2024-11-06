<?php

namespace App\Livewire\Runs;

use App\Actions\Github\DispatchSessionRun;
use App\Actions\Github\FetchSessionWorkflowRuns;
use App\Enums\SessionActivityType;
use App\Http\Controllers\SessionRunController;
use App\Models\Session;
use App\Models\SessionService;
use App\Models\SessionServiceRun;
use App\Models\SessionServiceSuite;
use App\Models\SessionServiceSuiteTest;
use Illuminate\Support\Collection;
use Livewire\Attributes\Computed;
use Livewire\Attributes\On;
use Livewire\Component;
use SimpleXMLElement;

class CreateManualTestRun extends Component
{
    public Session $session;
    public string $serviceId;
    public SessionService $service;
    public Collection $tests;

    public array $runIds = [];
    public int $index = 0;
    public array $result = [];
    public bool $open = false;


    protected $listeners = [
        'show' => 'show',
        'close' => 'close',
        'start-run' => 'startRun',
        'end-run' => 'submitTest'
    ];

    public function show()
    {
        $this->open = true;
    }

    public function close()
    {
        $this->open = false;
    }


    public function mount() {
        $this->service = SessionService::find($this->serviceId);
        $this->tests = $this->service->getManualTests();
    }

    public function startRun() {

        $run = $this->service->runs()->create([
            'service_id' => $this->service->id,
            'status' => 'In Progress', // Change this to fail if any test fails
            'started_at' => now(),
        ]);
        $this->runIds[] = $run->id;

        $this->session->activity()->create([
            'user_id' => auth()->id(),
            'type' => SessionActivityType::manual_run_started,
            'body' => "Manual run {$run->id} for {$this->service->displayName} started by " . auth()->user()->name  
        ]);
        $this->show();
    }

    public function next() {
        $this->goToIndex($this->index + 1);
    }

    public function prev() {
        $this->goToIndex($this->index - 1);
    }

    public function goToIndex(int $index) {
        // check if array index is within bounds
        if ($this->tests->has($index)) {
            $this->index = $index;
        }
    }

    #[Computed]
    public function currentTest()
    {
        return $this->tests->get($this->index);
    }

    public function updateResults(int $id, string $result, string $comment) {
        $test = SessionServiceSuiteTest::find($id);
        $this->result[$id] = [
            'test_id' => $test->id,
            'service_id' => $test->suite->service->id,
            'suite_id' => $test->suite->id,
            'status' => $result,
            'comment' => $comment,
        ];

        $this->updateResultLog();

        if ($this->index < $this->tests->count() - 1) {
            $this->next();
        } else {
            $this->submitTest();
        }
    }

    public function updateResultLog() {
        collect($this->runIds)->each(function($id) {
            $run = SessionServiceRun::find($id);

            $results = collect($this->result)
                ->filter(fn($result) => $result['service_id'] === $run->service_id);

            $xml = $this->transformResultLog($run, $results->toArray());

            $run->update([
                'result_log' => $xml,
                'passed' => count(collect($results)->where('status', 'pass')),
                'failed' => count($results->where('status', 'fail'))
            ]);
        });

    }

    public function transformResultLog(SessionServiceRun $run, array $data): string
    {
        $xml = new SimpleXMLElement('<?xml version="1.0" encoding="UTF-8"?><testsuites></testsuites>');

        collect($data)->groupBy('suite_id')
            ->each(function ($tests, $suiteId) use ($xml, $run) {
                $suite = SessionServiceSuite::findOrFail($suiteId);
                $testSuite = $xml->addChild('testsuite');
                $testSuite->addAttribute('name', explode('.', $suite->name)[0]);
                $testSuite->addAttribute('tests', count($tests));

                $testSuite->addAttribute('failures', count($tests->where('status', 'fail')));
                $testSuite->addAttribute('time', 0);

                collect($tests)->each(function($result) use ($xml, $testSuite) {
                    $test = SessionServiceSuiteTest::find($result['test_id']);

                    $testCase = $testSuite->addChild('testcase');
                    $testCase->addAttribute('id', $test->id);
                    $testCase->addAttribute('name', explode('.', $test->name)[0]);
                    $testCase->addAttribute('classname', $test->name);
                    $testCase->addAttribute('status', $test['status']);

                    if ($result['comment']) {
                        $testCase->addChild('system-out', $result['comment']);
                    }

                    if ($result['status'] === "fail") {
                        $failure = $testCase->addChild('failure', "Test failed");
                        $failure->addAttribute('message', $result['comment']);
                    }
                });
            });

        return $xml->asXML();
    }



    public function submitTest()
    {
        collect($this->runIds)->each(function($id) {
            $run = SessionServiceRun::find($id);

            $run->update([
               'status' => 'success',
                'finished_at' => now(),
            ]);

            $body = "Finished run {$run->id} for {$run->service->displayName}  with <span class='pass'>{$run->passed} passed</span> and <span class='fail'>{$run->failed} failed</span> tests";

            $this->session->activity()->create([
                'user_id' => auth()->id(),
                'type' => SessionActivityType::manual_run_finished,
                'body' => $body
            ]);
        });




        return redirect(request()->header('Referer'));
    }

    public function render()
    {
        return view('livewire.runs.create-manual-test-run');
    }
}
