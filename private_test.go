// Copyright (c) 2023 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package fx_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

func TestPrivateModule(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name    string
		getApp  func(bool) *App
		wantErr string
	}{
		{
			name: "Private Supply",
			getApp: func(private bool) *App {
				opts := []Option{Supply(5)}
				if private {
					opts = append(opts, Private)
				}
				return New(
					Module("SubModule", opts...),
					Invoke(func(a int) {}),
				)
			},
			wantErr: "missing type: int",
		},
		{
			name: "Deeply Nested Modules",
			getApp: func(private bool) *App {
				opts := []Option{
					Module("ModuleB",
						Module("ModuleC",
							Provide(func() string { return "s" }),
						),
					),
				}
				if private {
					opts = append(opts, Private)
				}
				return New(
					Module("ModuleA", opts...),
					Invoke(func(a string) {}),
				)
			},
			wantErr: "missing type: string",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Run("Public", func(t *testing.T) {
				app := tt.getApp(false)
				require.NoError(t, app.Err())
			})

			t.Run("Private", func(t *testing.T) {
				app := tt.getApp(true)
				err := app.Err()
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			})
		})
	}
}

func TestPrivateProvide(t *testing.T) {
	t.Parallel()

	t.Run("CanUsePrivateAndNonPrivateFromOuterModule", func(t *testing.T) {
		t.Parallel()

		app := fxtest.New(t,
			Module("SubModule", Invoke(func(a int, b string) {})),
			Provide(func() int { return 0 }, Private),
			Provide(func() string { return "" }),
		)
		app.RequireStart().RequireStop()
	})

	t.Run("CantUsePrivateFromSubModule", func(t *testing.T) {
		t.Parallel()

		app := NewForTest(t,
			Module("SubModule", Provide(func() int { return 0 }, Private)),
			Invoke(func(a int) {}),
		)
		err := app.Err()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing dependencies for function")
		assert.Contains(t, err.Error(), "missing type: int")
	})

	t.Run("DifferentModulesCanProvideSamePrivateType", func(t *testing.T) {
		t.Parallel()

		app := fxtest.New(t,
			Module("SubModuleA",
				Provide(func() int { return 1 }, Private),
				Invoke(func(s int) {
					assert.Equal(t, 1, s)
				}),
			),
			Module("SubModuleB",
				Provide(func() int { return 2 }, Private),
				Invoke(func(s int) {
					assert.Equal(t, 2, s)
				}),
			),
			Provide(func() int { return 3 }),
			Invoke(func(s int) {
				assert.Equal(t, 3, s)
			}),
		)
		app.RequireStart().RequireStop()
	})
}

func TestPrivateProvideWithDecorators(t *testing.T) {
	t.Parallel()

	t.Run("DecoratedPublicOrPrivateTypeInSubModule", func(t *testing.T) {
		t.Parallel()

		runApp := func(private bool) {
			provideOpts := []interface{}{func() int { return 0 }}
			if private {
				provideOpts = append(provideOpts, Private)
			}
			app := NewForTest(t,
				Module("SubModule",
					Provide(provideOpts...),
					Decorate(func(a int) int { return a + 2 }),
					Invoke(func(a int) { assert.Equal(t, 2, a) }),
				),
				Invoke(func(a int) { assert.Equal(t, 0, a) }),
			)
			err := app.Err()
			if private {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "missing dependencies for function")
				assert.Contains(t, err.Error(), "missing type: int")
			} else {
				require.NoError(t, err)
			}
		}

		t.Run("Public", func(t *testing.T) { runApp(false) })
		t.Run("Private", func(t *testing.T) { runApp(true) })
	})

	t.Run("DecoratedPublicOrPrivateTypeInOuterModule", func(t *testing.T) {
		t.Parallel()

		runApp := func(private bool) {
			provideOpts := []interface{}{func() int { return 0 }}
			if private {
				provideOpts = append(provideOpts, Private)
			}
			app := fxtest.New(t,
				Provide(provideOpts...),
				Decorate(func(a int) int { return a - 5 }),
				Invoke(func(a int) {
					assert.Equal(t, -5, a)
				}),
				Module("Child",
					Decorate(func(a int) int { return a + 10 }),
					Invoke(func(a int) {
						assert.Equal(t, 5, a)
					}),
				),
			)
			app.RequireStart().RequireStop()
		}

		t.Run("Public", func(t *testing.T) { runApp(false) })
		t.Run("Private", func(t *testing.T) { runApp(true) })
	})

	t.Run("CannotDecoratePrivateChildType", func(t *testing.T) {
		t.Parallel()

		app := NewForTest(t,
			Module("Child",
				Provide(func() int { return 0 }, Private),
			),
			Decorate(func(a int) int { return a + 5 }),
			Invoke(func(a int) {}),
		)
		err := app.Err()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing dependencies for function")
		assert.Contains(t, err.Error(), "missing type: int")
	})
}
