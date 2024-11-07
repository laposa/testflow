@php
    /** @var \App\Models\Session $session */
    /** @var array $selectedServices */
    /** @var bool $pollingEnabled */
@endphp

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

                    $relatedRuns = $run ? $run->service->runs()->where('type', $run->type)->where('id', '!=', $run->id)->get() : null;

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
                            @if ($run->result_log)
                                <x-passed-failed passed="{{ $run->passed }}"
                                                 failed="{{ $run->failed }}" />
                            @elseif ($run->type === 'manual')
                                <button type="button" class="filled"
                                        x-on:click.prevent="$dispatch('manual-test-run.load-run', { runId: {{ $run->id }}})"
                                >
                                    Complete Manual Tests
                                </button>
                            @endif
                        </td>
                        <td>
                            @if (count($relatedRuns) > 0)
                                <a href="#"
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
                                    <x-passed-failed passed="{{ $relatedRun->passed }}"
                                                     failed="{{ $relatedRun->failed }}" />
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


