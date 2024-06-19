<?php

namespace App\Models;

use App\Actions\Github\FetchSessionWorkflowRuns;
use Illuminate\Database\Eloquent\Casts\Attribute;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;

class Session extends Model
{
    protected $fillable = [
        'issuer_id',
        'name',
        'data',
    ];

    protected $table = 'test_sessions';

    protected function casts(): array
    {
        return [
            'data' => 'array'
        ];
    }

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

    public function runs(): HasMany
    {
        return $this->hasMany(SessionRun::class);
    }

    protected function workflowRuns(): Attribute
    {
        return new Attribute(
            get: fn () => app(FetchSessionWorkflowRuns::class)->handle($this)
        );
    }
}
