@php
    /**
     * @var \App\Models\Session $session
     * @var \Illuminate\Support\Collection $reviewRequest
     */
@endphp
<x-layout>
    <x-portal-section title="{{ $session->name }}">
        <div class="session-environment">
            Environment: <span>{{ ucfirst($session->environment) }}</span>
        </div>
        <livewire:runs.list-runs :session="$session" />
        <livewire:runs.manual-test-run  />
    </x-portal-section>

    <livewire:activity.activity-list :session="$session" />

    @foreach ($reviewRequests as $reviewRequest)
        @if ($reviewRequest && $reviewRequest->reviewer_id === auth()->id())
            <livewire:reviews.review-form :review-request="$reviewRequest" :session="$session" />
        @else
            <div class="review-form">
                Review pending for {{ $reviewRequest->reviewer->name }}
                @if ($reviewRequest && $reviewRequest->requester_id === auth()->id())
                    <br>
                    <livewire:reviews.revoke-review-form :review-request="$reviewRequest" />
                @endif
            </div>
        @endif
    @endforeach

    <div>
        <livewire:comments.create-comment :session="$session" />
    </div>
    <div>
        <livewire:reviews.request-review :session="$session" />
    </div>

    <section x-data="{ show: false }">
        <p>
            <a class="button" href="#" x-on:click.prevent="show = ! show">
                <span x-text="show ? 'Hide' : 'Show'"></span> selected tests
            </a>
        </p>
        <ul class="session-selected-suites" x-show="show">
            @foreach ($session->services as $service)
                <li><b>{{ $service->displayName }}</b></li>
                <ul>
                    @foreach ($service->suites as $suite)
                        <li><b>{{ $suite->name }}</b></li>
                        <ul>
                            @foreach ($suite->tests as $test)
                                <li>
                                    {{ $test->name }}

                                </li>
                            @endforeach
                        </ul>
                    @endforeach
                </ul>
            @endforeach
        </ul>
        <hr>
        <div class="session-report-actions">
            <a class="button" href="/sessions/{{ $session->id }}/executive">
                View Executive Report
            </a>
            <a class="button" href="/sessions/{{ $session->id }}/full">
                View Full Report
            </a>
        </div>
    </section>

</x-layout>
