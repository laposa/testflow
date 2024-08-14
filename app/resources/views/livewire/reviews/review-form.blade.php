@php
    /** @var App\Models\ReviewRequest $reviewRequest */
@endphp
<div class="review-form">
    <div class="review-form-icon">
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5"
            stroke="currentColor" class="size-6">
            <path stroke-linecap="round" stroke-linejoin="round"
                d="m9.75 9.75 4.5 4.5m0-4.5-4.5 4.5M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z" />
        </svg>
    </div>
    <div class="review-form-title">
        <h3>Review Required</h3>
        <p>{{ $reviewRequest->requester->name }} requested a review</p>
    </div>

    <div class="review-form-actions">
        <form wire:submit="approve">
            @csrf
            <button class="button filled" type="submit">Approve</button>

        </form>

        <form wire:submit="reject">
            @csrf
            <button class="button danger" type="submit">Reject</button>
        </form>
    </div>
</div>
