@php
    /**
     * @var \App\Models\Session $session
     * @var \Illuminate\Support\Collection $reviewRequest
     */
@endphp
<x-layout>
    <x-portal-section title="{{ $session->name }}">
        <livewire:runs.list-runs :session="$session" />
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
            <a class="button" href="#" x-on:click="show = ! show">
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
    <p><a class="button" href="#" onclick="print()">Print Executive
            Report</a></p>
</x-layout>
