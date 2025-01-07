<?php
namespace App\Livewire\Runs;

use App\Enums\SessionActivityType;
use App\Models\SessionService;
use App\Models\SessionServiceRun;
use App\Models\SessionServiceSuite;
use App\Models\SessionServiceSuiteTest;
use Illuminate\Support\Collection;
use Livewire\Attributes\Computed;
use Livewire\Attributes\On;
use Livewire\Component;
use SimpleXMLElement;

class ManualTestRun extends Component
{
    public SessionServiceRun|null $run = null;
    public SessionService $service;
    public Collection $tests;

    public array $result = [];
    public int $index = 0;

    protected $listeners = [
        'end-run' => 'submitTest'
    ];

    /**
     * Load the run and open the modal
     * To load a run call `$dispatch('manual-test-run.load-run', runId)` from a front-end component
     */
    #[On('manual-test-run.load-run')]
    public function loadRun(int $runId)
    {
        $this->index = 0;
        $this->run = SessionServiceRun::findOrFail($runId);
        $this->dispatch('open-modal', "manual-run");
    }

    public function next() {
        $this->goToIndex($this->index + 1);
    }

    public function prev() {
        $this->goToIndex($this->index - 1);
    }

    public function goToIndex(int $index) {
        if ($this->tests->has($index)) {
            $this->index = $index;
        }
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
            $this->dispatch('manual-test-next');
        } else {
            $this->submitTest();
        }
    }

    public function updateResultLog() {
        $results = collect($this->result)
            ->filter(fn($result) => $result['service_id'] === $this->run->service_id);

        $xml = $this->transformResultLog($this->run, $results->toArray());

        $this->run->update([
            'result_log' => $xml,
            'passed' => count(collect($results)->where('status', 'pass')),
            'failed' => count($results->where('status', 'fail'))
        ]);
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
                        $testCase->addChild('system-out', htmlspecialchars($result['comment']));
                    }

                    if ($result['status'] === "fail") {
                        $failure = $testCase->addChild('failure', "Test failed");
                        $failure->addAttribute('message', htmlspecialchars($result['comment']));
                    }
                });
            });

        return $xml->asXML();
    }



    public function submitTest()
    {
            $this->run->update([
                'status' => 'success',
                'finished_at' => now(),
            ]);

            $body = "Finished run {$this->run->id} for {$this->run->service->displayName}  with <span class='pass'>{$this->run->passed} passed</span> and <span class='fail'>{$this->run->failed} failed</span> tests";

            $this->run->service->session->activity()->create([
                'user_id' => auth()->id(),
                'type' => SessionActivityType::manual_run_finished,
                'body' => $body
            ]);

        $this->dispatch('list-runs.refresh');

        // Since we are overriding the close event in the render method
        // to show a confirm dialog we use this custom event to silently close the modal
        $this->dispatch('silent-close-modal', "manual-run");
    }

    #[Computed]
    public function currentTest()
    {
        return $this->tests->get($this->index);
    }
    public function render()
    {
        if ($this->run) {
            $this->service = $this->run->service;
            $this->tests = $this->service->getManualTests();
        }

        return <<<'HTML'
        <x-dialog name="manual-run"
            x-on:close-modal.window="
                if (confirm('Closing will finish the run, would you like to continue?')) {
                    $dispatch('end-run');
                    $event.detail == 'manual-run' ? close('manual-run') : null
                }
            "
            x-on:silent-close-modal.window="$event.detail == 'manual-run' ? close('manual-run') : null"
        >
            @if ($run)
                <div>
                    <div>Test {{ $index + 1}}/{{$tests->count()}}</div>
                    @if ($this->currentTest)
                        <livewire:runs.manual-test-form
                            key="{{$this->currentTest->id}}"
                            :test="$this->currentTest"
                            @updated="updateResults($event.detail.id, $event.detail.result, $event.detail.comment)"/>
                    @else
                        No test selected
                    @endif
                </div>
            @else
                No Run Selected
            @endif
        </x-dialog>
        HTML;
    }
}
