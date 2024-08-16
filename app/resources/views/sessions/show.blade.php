@php
    /**
     * @var \App\Models\Session $session
     * @var \Illuminate\Support\Collection $activity
     * @var \App\Models\ReviewRequest $reviewRequest
     * @var \App\Models\SessionRun $latestRun
     */
@endphp
<x-layout>
    <x-portal-section title="{{ $session->name }}">
        <livewire:runs.list-runs :session="$session" />
    </x-portal-section>

    @if ($activity && $activity->count() > 0)
        <div class="activity-feed">
            @foreach ($activity as $entry)
                <livewire:activity.activity-feed-item :entry="$entry" />
            @endforeach
        </div>

        @foreach ($reviewRequests as $reviewRequest)
            @if ($reviewRequest && $reviewRequest->reviewer_id === auth()->id())
                <livewire:reviews.review-form :review-request="$reviewRequest" :model="$latestRun" />
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
    @endif

    @if ($latestRun)
        <div>
            <livewire:comments.create-comment :model="$latestRun" />
        </div>
        <div>
            <livewire:reviews.request-review :model="$latestRun" />
        </div>
    @endif

    <p style="text-align: right"><a class="button" href="#" onclick="print()">Print Executive
            Report</a></p>
    <section x-data="{ show: false }">
        <p>
            <a href="#" x-on:click="show = ! show">
                <span x-text="show ? 'Hide' : 'Show'"></span> selected tests
            </a>
        </p>
        <ul class="session-selected-suites" x-show="show">
            @foreach ($session->itemsGrouped as $path => $tests)
                <li><b>{{ getTestSuiteName($tests[0]) }}</b></li>
                <ul>
                    @foreach ($tests as $test)
                        <li>{{ $test['test_name'] }}</li>
                    @endforeach
                </ul>
            @endforeach
        </ul>
    </section>
</x-layout>
