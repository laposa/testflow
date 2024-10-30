<?php

namespace App\Models;

use App\Actions\Github\FetchManualTestContent;
use App\Data\ManualTestData;
use Illuminate\Database\Eloquent\Casts\Attribute;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class SessionServiceSuiteTest extends Model
{
    protected $fillable = ['name', 'path'];

    protected $table = 'test_session_service_suite_tests';

    public function suite(): BelongsTo
    {
        return $this->belongsTo(SessionServiceSuite::class, 'suite_id');
    }

    public function isManualTest(): Attribute
    {
        return new Attribute(get: fn() => \Str::contains($this->name, "manual"));
    }

    public function getInstructions(): ManualTestData | null
    {
        return \Cache::remember(
            "{$this->id}-instructions",
            now()->addMinutes(60),
            fn () => $this->isManualTest ? app(FetchManualTestContent::class)->handle($this) : null
        );
    }

}
