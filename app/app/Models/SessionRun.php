<?php

namespace App\Models;

use App\Services\JUnitLogParser;
use Illuminate\Database\Eloquent\Casts\Attribute;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\BelongsToMany;
use Illuminate\Database\Eloquent\Relations\MorphMany;

class SessionRun extends Model
{
    protected $fillable = ['service_name', 'status', 'passed', 'failed', 'result_log', 'run_log'];

    protected $table = 'test_session_runs';

    public function session(): BelongsTo
    {
        return $this->belongsTo(Session::class);
    }

    public function items(): BelongsToMany
    {
        return $this->belongsToMany(
            SessionItem::class,
            'test_session_items_test_session_runs',
        )->withTimestamps();
    }

    public function comments(): MorphMany
    {
        return $this->morphMany(Comment::class, 'commentable');
    }

    public function reviewRequests(): MorphMany
    {
        return $this->morphMany(ReviewRequest::class, 'reviewable');
    }

    public function itemsGrouped(): Attribute
    {
        return new Attribute(
            get: fn() => $this->items->groupBy(
                fn($item) => $item->repository_name . $item->service_name . $item->suite_name,
            ),
        );
    }

    public function parsedResults(): Attribute
    {
        return new Attribute(get: fn() => new JUnitLogParser($this->result_log));
    }
}
