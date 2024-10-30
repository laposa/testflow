<?php

namespace App\Livewire\Runs;

use App\Actions\Github\DispatchSessionRun;
use App\Actions\Github\FetchSessionWorkflowRuns;
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
    public Collection $tests;
    public int $index = 0;
    public array $result = [];
    public bool $open = false;


    protected $listeners = [
        'show' => 'show',
        'close' => 'close'
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
        $this->tests = $this->session->getManualTests();
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

    #[On('test-updated')]
    public function updateResults(int $id, string $result, string $comment) {
        $test = SessionServiceSuiteTest::find($id);
        $this->result[$id] = [
            'test_id' => $test->id,
            'service_id' => $test->suite->service->id,
            'suite_id' => $test->suite->id,
            'status' => $result,
            'comment' => $comment,
        ];

        if ($this->index < $this->tests->count() - 1) {
            $this->next();
        } else {
            $this->submitTest();
        }
    }

    public function submitTest()
    {
        collect($this->result)
            ->groupBy('service_id')
            ->each(function ($tests, $serviceId) {
                $service = SessionService::find($serviceId);
                $xml = new SimpleXMLElement('<?xml version="1.0" encoding="UTF-8"?><testsuites></testsuites>');

                collect($tests)
                    ->groupBy('suite_id')
                    ->each(function ($tests, $suiteId) use ($xml, $service){
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


                           if ($result['status'] === "fail") {
                               $failure = $testCase->addChild('failure', "Test failed");
                               $failure->addAttribute('message', $result['comment']);
                           }
                        });
                    }
                );

                $service->runs()->create([
                    'service_id' => $service->id,
                    'status' => 'success', // Change this to fail if any test fails
                    'result_log' => $xml->asXML(),
                    'run_log' => '',
                    'finished_at' => now(),
                    'started_at' => now(),
                    'passed' => count($tests->where('status', 'pass')),
                    'failed' => count($tests->where('status', 'fail')),
                ]);

                return redirect(request()->header('Referer'));
            }
        );


    }

    public function render()
    {
        return view('livewire.runs.create-manual-test-run');
    }
}
