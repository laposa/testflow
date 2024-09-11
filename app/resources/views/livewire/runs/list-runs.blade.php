@php
    /** @var \App\Models\Session $session */
@endphp

<div @if ($pollingEnabled) wire:poll.10s @endif x-data="{ selectedServices: [];displayedRuns: [] }">
    <table class="runs-list">
        <thead>
            <tr>
                <th></th>
                <th>Service</th>
                <th>Run ID</th>
                <th>Started</th>
                <th>Duration</th>
                <th>Status</th>
                <th>Results</th>
                <th></th>
            </tr>
        </thead>
        <tbody>
            @foreach ($session->services as $service)
                @php
                    /** @var \App\Models\SessionService $service */
                    $run = $service->runs->first();
                @endphp
                <tr>
                    <td>
                        <input type="checkbox" wire:model="selectedServices"
                            value="{{ $service->id }}">
                    </td>
                    <td>{{ $service->displayName }}</td>
                    @if (count($service->runs) == 0)
                        <td>No runs</td>
                        <td>-</td>
                        <td>-</td>
                        <td>-</td>
                        <td>-</td>
                    @else
                        <td>
                            @if ($run->run_log)
                                <a href="/sessions/{{ $session->id }}/run/{{ $run->id }}"
                                    title="Show logs">{{ $run->id }}</a>
                            @else
                                {{ $run->id }}
                            @endif
                        </td>
                        <td>{{ $run->created_at }}</td>
                        <td>
                            @if ($run->result_log)
                                {{ $run->humanReadableDuration }}
                            @endif
                        </td>
                        <td>
                            <span class="{{ $run->status }}">{{ $run->status }}</span>
                        </td>
                        <td>
                            @if ($run->result_log)
                                <span class="pass">{{ $run->parsedResults->getTotalPassed() }}
                                    passed</span> and <span
                                    class="fail">{{ $run->parsedResults->getTotalFailures() }}
                                    failed</span>
                            @endif
                        </td>
                        <td>
                            @if (count($service->runs) > 1)
                                <a href="#"
                                    x-on:click.prevent="$wire.displayedRuns.includes({{ $service->id }}) ? $wire.displayedRuns = $wire.displayedRuns.filter(id => id !== {{ $service->id }}) : $wire.displayedRuns.push({{ $service->id }})">

                                    <img src="/images/icons/plus.svg" alt="Show more runs"
                                        x-show="!$wire.displayedRuns.includes({{ $service->id }})"
                                        class="icon">

                                    <img src="/images/icons/minus.svg" alt="Hide runs"
                                        x-show="$wire.displayedRuns.includes({{ $service->id }})"
                                        class="icon">
                                </a>
                            @endif
                        </td>
                    @endif
                </tr>

                @if (count($service->runs) > 1)
                    @foreach ($service->runs->skip(1) as $run)
                        <tr class="more-runs-list"
                            x-show="$wire.displayedRuns.includes({{ $service->id }})">
                            <td></td>
                            <td></td>
                            <td>
                                @if ($run->run_log)
                                    <a href="/sessions/{{ $session->id }}/run/{{ $run->id }}"
                                        title="Show logs">{{ $run->id }}</a>
                                @else
                                    {{ $run->id }}
                                @endif
                            </td>
                            <td>{{ $run->created_at }}</td>
                            <td>
                                @if ($run->result_log)
                                    {{ $run->humanReadableDuration }}
                                @endif
                            </td>
                            <td>
                                <span class="{{ $run->status }}">{{ $run->status }}</span>
                            </td>
                            <td>
                                @if ($run->result_log)
                                    <span
                                        class="pass">{{ $run->parsedResults->getTotalPassed() }}
                                        passed</span> and <span
                                        class="fail">{{ $run->parsedResults->getTotalFailures() }}
                                        failed</span>
                                @endif
                            </td>
                            <td></td>
                        </tr>
                    @endforeach
                @endif
            @endforeach
        </tbody>
    </table>

    <button type="button" wire:click="createRun" wire:loading.attr="disabled" class="filled">
        Run <span
            x-text="$wire.selectedServices.length === 0 ? 'All Tests' : 'Selected Tests (' + $wire.selectedServices.length + ')'"></span>
    </button>
</div>
