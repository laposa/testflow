@php
    /** @var \App\Models\Session $session */
    /** @var array $selectedServices */
    /** @var bool $pollingEnabled */
@endphp

<div
    @if ($pollingEnabled) wire:poll.10s @endif
    x-data="{  displayedRuns: [] }">
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
                    $run = $service->runs->first();
                @endphp
                <tr>
                    <td>
                        <input type="checkbox"
                               wire:model.live="selectedServices"
                            value="{{ $service->id }}">
                    </td>
                    <td>{{ $service->displayName }}</td>
                    @if (count($service->runs) == 0)
                        <td>No runs</td>
                        <td>-</td>
                        <td>-</td>
                        <td>-</td>
                    @else
                        <td>
                            <a href="/sessions/{{ $session->id }}/run/{{ $run->id }}"
                                title="Show logs">{{ $run->id }}</a>
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
                                    <a href="/sessions/{{ $session->id }}/run/{{ $run->id }}"
                                        title="Show logs">{{ $run->id }}</a>
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
            Run <span
                x-text="$wire.selectedServices.length === 0 ? 'All Tests' : 'Selected Tests (' + $wire.selectedServices.length + ')'"></span>
        </button>

        @if ($session->hasManualTests)
            @php
                $manualServices = count($this->selectedServices) > 0 ?
                                    $this->selectedServices :
                                    $session->services->filter(fn($service) => $service->hasManualTests())->pluck('id')->toArray();
            @endphp
            @foreach($manualServices as $service)
                <livewire:runs.create-manual-test-run key="manual-service-{{$service}}" :session="$session" :service-id="$service" />
            @endforeach
        @endif

    </div>
</div>
