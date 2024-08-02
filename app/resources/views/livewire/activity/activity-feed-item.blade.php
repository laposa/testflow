@php
/** @var App\Models\Comment|App\Models\ReviewRequest $entry */
@endphp

    @if ($entry instanceof App\Models\Comment)
        @php
            /** @var App\Models\Comment $entry */
        @endphp
        <div class="activity-feed-item activity-feed-item--comment">
            <div>
                <div class="activity-feed-item-header">
                    <div>
                        <strong>{{ $entry->user->name }}</strong> commented {{ $entry->created_at->diffForHumans() }}
                    </div>
                </div>
                <div class="activity-feed-item-body">{{ $entry->body }}</div>
            </div>
        </div>
    @endif
    @if ($entry instanceof App\Models\ReviewRequest)
        @php
            /** @var App\Models\ReviewRequest $entry */
        @endphp
        <div class="activity-feed-item activity-feed-item--review-request">
            @if ($entry->status === "pending")
                {{ $entry->requester->name }} requested a review for {{ $entry->reviewer->name }}

                <form wire:submit="delete">
                    @csrf
                    <button type="submit">Revoke</button>
                </form>
            @endif
            @if ($entry->status === "approved")
                {{ $entry->reviewer->name }} approved the review request from {{ $entry->requester->name }}
            @endif
            @if ($entry->status === "rejected")
                {{ $entry->reviewer->name }} rejected the review request from {{ $entry->requester->name }}
            @endif
                <div>{{ $entry->created_at->diffForHumans() }}</div>

        </div>
    @endif
