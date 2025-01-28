@php
    /** @var \App\Models\Session $session */
    /** @var array $selectedServices */
    /** @var bool $pollingEnabled */
@endphp
@script
<script type="text/javascript">

    document.addEventListener('change', function (e) {
        //manual test fail/pass control
        if(e.target.closest('.custom-radio')) {
            let checks = [...document.querySelectorAll('.manual-test .custom-radio .radio-pass')];
            let fails = [...document.querySelectorAll('.manual-test .custom-radio .radio-fail')];
            
            if(checks.length > 0 && checks.every(check => check.checked)) {
                markTestAs('pass');
            }
    
            if(fails.length > 0 && fails.some(fail => fail.checked)) {
                markTestAs('fail');
            }
        }

        if(e.target.closest('.test-status')) {
            document.querySelector('.manual-test button.submit').disabled = false;
        }
    });

    Livewire.on('manual-test-next', (event) => {
        document.querySelector('.modal-content').scrollTop = 0;
    });

    function markTestAs(status) {
        switch(status) {
            case 'pass':
                document.querySelector('.manual-test .test-status #pass').checked = true;
                document.querySelector('.manual-test .test-status #pass').dispatchEvent(new Event('change'));
                break;
            case 'fail':
                document.querySelector('.manual-test .test-status #fail').checked = true;
                document.querySelector('.manual-test .test-status #fail').dispatchEvent(new Event('change'));
                break;
        }
        document.querySelector('.manual-test button.submit').disabled = false;
    }
</script>
@endscript
<div
    @if ($pollingEnabled) wire:poll.30s="refreshSessionWorkflowRuns(false)" @endif
    x-data="{ expanded: [] }">
    <table class="runs-list">
        <thead>
            <tr>
                <th></th>
                <th>Service</th>
                <th>Run ID</th>
                <th>Initiated</th>
                <th>Duration</th>
                <th>Results</th>
                <th></th>
            </tr>
        </thead>
        <tbody>
            @foreach ($session->services as $service)
                @php
                    /** @var \App\Models\SessionService $service */
                    $run = $service->runs()
                        ->latest()
                        ->first();

                    $relatedRuns = $run ? $run->service->runs()->where('type', $run->type)->where('id', '!=', $run->id)->latest()->get() : null;
                @endphp
                <tr>
                    <td>
                        <input type="checkbox"
                               wire:model.live="selectedServices"
                               value="{{ $service->id }}"
                        >
                    </td>
                    <td>
                        {{ $service->display_name }}

                    </td>
                    @if (count($service->runs) == 0)
                        <td>No runs</td>
                        <td>-</td>
                        <td>-</td>
                        <td>-</td>
                    @else
                        <td>
                            <a href="/sessions/{{ $session->id }}/run/{{ $run->id }}"
                               title="Show logs">
                                {{ $run->id }}
                            </a>
                        </td>
                        <td>{{ $run->created_at }}</td>
                        <td
                            @if ($run->started_at) title="Picked by runner at {{ $run->started_at }}" @endif>
                            @if ($run->finished_at)
                                {{ $run->humanReadableDuration }}
                            @else
                                <span class="running">Running</span>
                            @endif
                        </td>
                        <td>
                            @if ($run->result_log && $run->finished_at)
                                <x-passed-failed 
                                    passed="{{ $run->passed }}"
                                    failed="{{ $run->failed }}" 
                                    skipped="{{ $run->skipped }}"
                                />
                            @elseif ($run->type === 'manual')
                                <button type="button" class="filled"
                                        x-on:click.prevent="$dispatch('manual-test-run.load-run', { runId: {{ $run->id }}})"
                                >
                                    Complete manual tests
                                </button>
                                <button type="button"
                                        x-on:click.prevent="$dispatch('manual-test-run.mark-as-finished', { runId: {{ $run->id }}})"
                                >
                                    Mark run as finished
                                </button>
                            @endif
                        </td>
                        <td>
                            @if (count($relatedRuns) > 0)
                                <a href="#" class="expand-icon"
                                   x-on:click.prevent="expanded.includes('{{ $run->type }}-{{ $service->id }}') ? expanded = expanded.filter(id => id !== '{{ $run->type }}-{{ $service->id }}') : expanded.push('{{ $run->type }}-{{ $service->id }}')">

                                    <img src="/images/icons/plus.svg" alt="Show more runs"
                                         x-show="!expanded.includes('{{ $run->type }}-{{ $service->id }}')"
                                         class="icon">

                                    <img src="/images/icons/minus.svg" alt="Hide runs"
                                         x-cloak
                                         x-show="expanded.includes('{{ $run->type }}-{{ $service->id }}')"
                                         class="icon">
                                </a>
                            @endif
                        </td>
                    @endif
                </tr>
                @if ($relatedRuns && count($relatedRuns) > 0)
                    @foreach ($relatedRuns as $relatedRun)
                        <tr class="more-runs-list"
                            x-cloak
                            x-show="expanded.includes('{{ $run->type }}-{{ $service->id }}')">
                            <td></td>
                            <td></td>
                            <td>
                                <a href="/sessions/{{ $session->id }}/run/{{ $relatedRun->id }}"
                                   title="Show logs">{{ $relatedRun->id }}</a>
                            </td>
                            <td>{{ $relatedRun->created_at }}</td>
                            <td
                                @if ($relatedRun->started_at) title="Picked by runner at {{ $relatedRun->started_at }}" @endif>
                                @if ($relatedRun->finished_at)
                                    {{ $relatedRun->humanReadableDuration }}
                                @else
                                    <span class="running">Running</span>
                                @endif
                            </td>
                            <td>
                                @if ($relatedRun->result_log)
                                    <x-passed-failed 
                                        passed="{{ $relatedRun->passed }}"
                                        failed="{{ $relatedRun->failed }}" 
                                        skipped="{{ $relatedRun->skipped }}"
                                    />
                                @endif
                            </td>
                            <td></td>
                        </tr>
                    @endforeach
                @endif
            @endforeach
        </tbody>
    </table>

    <div class="list-runs-actions">
        <button type="button" wire:click="createRun" wire:loading.attr="disabled" class="filled">
            @if (count($selectedServices) > 0)
                Run Selected Tests ({{ count($selectedServices) }})
            @else
                Run All Tests
            @endif
        </button>
    </div>
</div>


