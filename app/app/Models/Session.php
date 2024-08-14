<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Casts\Attribute;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;

class Session extends Model
{
    protected $fillable = ['issuer_id', 'environment', 'name'];

    protected $table = 'test_sessions';


    public function installation(): BelongsTo
    {
        return $this->belongsTo(Installation::class);
    }

    public function issuer(): BelongsTo
    {
        return $this->belongsTo(User::class, foreignKey: 'issuer_id');
    }

    public function items(): HasMany
    {
        return $this->hasMany(SessionItem::class);
    }

    public function itemsGrouped(): Attribute
    {
        return new Attribute(
            get: fn() => $this->items->groupBy(
                fn($item) => $item->repository_name . $item->service_name . $item->suite_name,
            ),
        );
    }

    public function itemsGroupedByService(): Attribute
    {
        return new Attribute(
            get: fn() => $this->items->groupBy(
                fn($item) => $item->repository_name . $item->service_name,
            ),
        );
    }

    public function runs(): HasMany
    {
        return $this->hasMany(SessionRun::class);
    }

    public function canRunTests(): bool
    {
        /** @var SessionRun|null $lastRun */
        // If there is a pending or rejected review request in the last test, we can't run tests
        $lastRun = $this->runs()->orderBy('created_at', 'desc')->first();
        if ($lastRun && $lastRun->reviewRequests()->orderBy('created_at', 'desc')->first()?->status !== "approved") {
            return false;
        }
        return true;
    }

    public function lastRun(): Attribute
    {
        return new Attribute(
            get: fn() => $this->runs()->orderBy('created_at', 'desc')->first(),
        );
    }

    public function isRunning(): Attribute
    {
        return new Attribute(
            get: fn() => $this->last_run && !$this->last_run?->result_log,
        );
    }

    public function passedCount(): Attribute
    {
        return new Attribute(
            get: fn() => $this->last_run?->parsedResults->getTotalPassed() ?? 0,
        );
    }

    public function failedCount(): Attribute
    {
        return new Attribute(
            get: fn() => $this->last_run?->parsedResults->getTotalFailures() ?? 0,
        );
    }

    public function status(): Attribute
    {
        return new Attribute(
            get: fn() => $this->last_run?->status ?? 'unknown',
        );
    }
}
